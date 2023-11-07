module Page.Status exposing (Model, Msg, init, subscriptions, update, view)

import Data.Metrics exposing (Metrics)
import Data.ServerConfig exposing (ServerConfig)
import Data.Session exposing (Session)
import DateFormat.Relative as Relative
import Effect exposing (Effect)
import Filesize
import Html
    exposing
        ( Html
        , div
        , h1
        , h2
        , i
        , text
        )
import Html.Attributes exposing (class)
import HttpUtil
import Sparkline as Spark
import Svg.Attributes as SvgAttrib
import Time exposing (Posix)



-- MODEL --


type alias Model =
    { session : Session
    , now : Posix
    , config : Maybe ServerConfig
    , metrics : Maybe Metrics
    , xCounter : Float
    , sysMem : Metric
    , heapSize : Metric
    , heapUsed : Metric
    , heapObjects : Metric
    , goRoutines : Metric
    , webSockets : Metric
    , smtpConnOpen : Metric
    , smtpConnTotal : Metric
    , smtpReceivedTotal : Metric
    , smtpErrorsTotal : Metric
    , smtpWarnsTotal : Metric
    , retentionDeletesTotal : Metric
    , retainedCount : Metric
    , retainedSize : Metric
    }


type alias Metric =
    { label : String
    , value : Int
    , formatter : Int -> String
    , graph : Spark.DataSet -> Html Msg
    , history : Spark.DataSet
    , minutes : Int
    }


init : Session -> ( Model, Effect Msg )
init session =
    ( { session = session
      , now = Time.millisToPosix 0
      , config = Nothing
      , metrics = Nothing
      , xCounter = 60
      , sysMem = Metric "System Memory" 0 Filesize.format graphZero initDataSet 10
      , heapSize = Metric "Heap Size" 0 Filesize.format graphZero initDataSet 10
      , heapUsed = Metric "Heap Used" 0 Filesize.format graphZero initDataSet 10
      , heapObjects = Metric "Heap # Objects" 0 fmtInt graphZero initDataSet 10
      , goRoutines = Metric "Goroutines" 0 fmtInt graphZero initDataSet 10
      , webSockets = Metric "Open WebSockets" 0 fmtInt graphZero initDataSet 10
      , smtpConnOpen = Metric "Open Connections" 0 fmtInt graphZero initDataSet 10
      , smtpConnTotal = Metric "Total Connections" 0 fmtInt graphChange initDataSet 60
      , smtpReceivedTotal = Metric "Messages Received" 0 fmtInt graphChange initDataSet 60
      , smtpErrorsTotal = Metric "Message Errors" 0 fmtInt graphChange initDataSet 60
      , smtpWarnsTotal = Metric "Message Warnings" 0 fmtInt graphChange initDataSet 60
      , retentionDeletesTotal = Metric "Retention Deletes" 0 fmtInt graphChange initDataSet 60
      , retainedCount = Metric "Stored Messages" 0 fmtInt graphZero initDataSet 60
      , retainedSize = Metric "Store Size" 0 Filesize.format graphZero initDataSet 60
      }
    , Effect.batch
        [ Effect.posixTime Tick
        , Effect.getServerConfig ServerConfigLoaded
        ]
    )


initDataSet : Spark.DataSet
initDataSet =
    List.range 0 59
        |> List.map (\x -> ( toFloat x, 0 ))



-- SUBSCRIPTIONS --


subscriptions : Model -> Sub Msg
subscriptions _ =
    Time.every (10 * 1000) Tick



-- UPDATE --


type Msg
    = MetricsReceived (Result HttpUtil.Error Metrics)
    | ServerConfigLoaded (Result HttpUtil.Error ServerConfig)
    | Tick Posix


update : Msg -> Model -> ( Model, Effect Msg )
update msg model =
    case msg of
        MetricsReceived (Ok metrics) ->
            ( updateMetrics metrics model, Effect.none )

        MetricsReceived (Err err) ->
            ( model, Effect.showFlash (HttpUtil.errorFlash err) )

        ServerConfigLoaded (Ok config) ->
            ( { model | config = Just config }, Effect.none )

        ServerConfigLoaded (Err err) ->
            ( model, Effect.showFlash (HttpUtil.errorFlash err) )

        Tick time ->
            ( { model | now = time }
            , Effect.getServerMetrics MetricsReceived
            )


{-| Update all metrics in Model; increment xCounter.
-}
updateMetrics : Metrics -> Model -> Model
updateMetrics metrics model =
    let
        x =
            model.xCounter
    in
    { model
        | metrics = Just metrics
        , xCounter = x + 1
        , sysMem = updateLocalMetric model.sysMem x metrics.sysMem
        , heapSize = updateLocalMetric model.heapSize x metrics.heapSize
        , heapUsed = updateLocalMetric model.heapUsed x metrics.heapUsed
        , heapObjects = updateLocalMetric model.heapObjects x metrics.heapObjects
        , goRoutines = updateLocalMetric model.goRoutines x metrics.goRoutines
        , webSockets = updateLocalMetric model.webSockets x metrics.webSockets
        , smtpConnOpen = updateLocalMetric model.smtpConnOpen x metrics.smtpConnOpen
        , smtpConnTotal =
            updateRemoteTotal
                model.smtpConnTotal
                metrics.smtpConnTotal
                metrics.smtpConnHist
        , smtpReceivedTotal =
            updateRemoteTotal
                model.smtpReceivedTotal
                metrics.smtpReceivedTotal
                metrics.smtpReceivedHist
        , smtpErrorsTotal =
            updateRemoteTotal
                model.smtpErrorsTotal
                metrics.smtpErrorsTotal
                metrics.smtpErrorsHist
        , smtpWarnsTotal =
            updateRemoteTotal
                model.smtpWarnsTotal
                metrics.smtpWarnsTotal
                metrics.smtpWarnsHist
        , retentionDeletesTotal =
            updateRemoteTotal
                model.retentionDeletesTotal
                metrics.retentionDeletesTotal
                metrics.retentionDeletesHist
        , retainedCount =
            updateRemoteMetric
                model.retainedCount
                metrics.retainedCount
                metrics.retainedCountHist
        , retainedSize =
            updateRemoteMetric
                model.retainedSize
                metrics.retainedSize
                metrics.retainedSizeHist
    }


{-| Update a single Metric, with history tracked locally.
-}
updateLocalMetric : Metric -> Float -> Int -> Metric
updateLocalMetric metric x value =
    { metric
        | value = value
        , history =
            Maybe.withDefault [] (List.tail metric.history)
                ++ [ ( x, toFloat value ) ]
    }


{-| Update a single Metric, with history tracked on server.
-}
updateRemoteMetric : Metric -> Int -> List Int -> Metric
updateRemoteMetric metric value history =
    { metric
        | value = value
        , history =
            history
                |> zeroPadList
                |> List.indexedMap (\x y -> ( toFloat x, toFloat y ))
    }


{-| Update a single Metric, with history tracked on server. Sparkline will chart changes to the
total instead of its absolute value.
-}
updateRemoteTotal : Metric -> Int -> List Int -> Metric
updateRemoteTotal metric value history =
    { metric
        | value = value
        , history =
            history
                |> zeroPadList
                |> changeList
                |> List.indexedMap (\x y -> ( toFloat x, toFloat y ))
    }



-- VIEW --


view : Model -> { title : String, modal : Maybe (Html msg), content : List (Html Msg) }
view model =
    { title = "Inbucket Status"
    , modal = Nothing
    , content =
        [ h1 [] [ text "Status" ]
        , div [] (configPanel model.config :: metricPanels model)
        ]
    }


configPanel : Maybe ServerConfig -> Html Msg
configPanel maybeConfig =
    let
        mailboxCap config =
            case config.storageConfig.mailboxMsgCap of
                0 ->
                    "Unlimited"

                cap ->
                    String.fromInt cap ++ " messages per mailbox"

        retentionPeriod config =
            case config.storageConfig.retentionPeriod of
                "" ->
                    "Forever"

                period ->
                    period
    in
    case maybeConfig of
        Nothing ->
            text "Loading server config..."

        Just config ->
            framePanel "Configuration"
                "fa-cog"
                [ textEntry "Version" (config.version ++ ", built on " ++ config.buildDate)
                , textEntry "SMTP Listener" config.smtpConfig.addr
                , textEntry "POP3 Listener" config.pop3Listener
                , textEntry "HTTP Listener" config.webListener
                , textEntry "Origin Policy" (originPolicy config)
                , textEntry "Destination Policy" (acceptPolicy config)
                , textEntry "Store Policy" (storePolicy config)
                , textEntry "Store Type" config.storageConfig.storeType
                , textEntry "Message Cap" (mailboxCap config)
                , textEntry "Retention Period" (retentionPeriod config)
                ]


acceptPolicy : ServerConfig -> String
acceptPolicy config =
    if config.smtpConfig.defaultAccept then
        "Allow all domains"
            ++ (case config.smtpConfig.rejectDomains of
                    Nothing ->
                        ""

                    Just [] ->
                        ""

                    Just domains ->
                        ", except: " ++ String.join ", " domains
               )

    else
        "Reject all domains"
            ++ (case config.smtpConfig.acceptDomains of
                    Nothing ->
                        ""

                    Just [] ->
                        ""

                    Just domains ->
                        ", except to: " ++ String.join ", " domains
               )


originPolicy : ServerConfig -> String
originPolicy config =
    "Allow all domains"
        ++ (case config.smtpConfig.rejectOriginDomains of
                Nothing ->
                    ""

                Just [] ->
                    ""

                Just domains ->
                    ", except from: " ++ String.join ", " domains
           )


storePolicy : ServerConfig -> String
storePolicy config =
    if config.smtpConfig.defaultStore then
        "All destination domains"
            ++ (case config.smtpConfig.discardDomains of
                    Nothing ->
                        ""

                    Just [] ->
                        ""

                    Just domains ->
                        ", except to: " ++ String.join ", " domains
               )

    else
        "No destination domains"
            ++ (case config.smtpConfig.storeDomains of
                    Nothing ->
                        ""

                    Just [] ->
                        ""

                    Just domains ->
                        ", except: " ++ String.join ", " domains
               )


metricPanels : Model -> List (Html Msg)
metricPanels model =
    case model.metrics of
        Nothing ->
            [ text "Loading metrics..." ]

        Just metrics ->
            [ framePanel "General Metrics"
                "fa-tachometer-alt"
                [ textEntry "Uptime" <|
                    "Started "
                        ++ Relative.relativeTime model.now metrics.startTime
                , viewMetric model.sysMem
                , viewMetric model.heapSize
                , viewMetric model.heapUsed
                , viewMetric model.heapObjects
                , viewMetric model.goRoutines
                , viewMetric model.webSockets
                ]
            , framePanel "SMTP Metrics"
                "fa-envelope"
                [ viewMetric model.smtpConnOpen
                , viewMetric model.smtpConnTotal
                , viewMetric model.smtpReceivedTotal
                , viewMetric model.smtpErrorsTotal
                , viewMetric model.smtpWarnsTotal
                ]
            , framePanel "Storage Metrics"
                "fa-archive"
                [ textEntry "Retention Scan" (retentionScan model)
                , viewMetric model.retentionDeletesTotal
                , viewMetric model.retainedCount
                , viewMetric model.retainedSize
                ]
            ]


retentionScan : Model -> String
retentionScan model =
    case ( model.config, model.metrics ) of
        ( Just config, Just metrics ) ->
            case config.storageConfig.retentionPeriod of
                "" ->
                    "Disabled"

                _ ->
                    case Time.posixToMillis metrics.scanCompleted of
                        0 ->
                            "Not completed"

                        _ ->
                            "Completed " ++ Relative.relativeTime model.now metrics.scanCompleted

        ( _, _ ) ->
            "No data"


textEntry : String -> String -> Html Msg
textEntry name value =
    div [ class "metric" ]
        [ div [ class "label" ] [ text name ]
        , div [ class "text-value" ] [ text value ]
        ]


viewMetric : Metric -> Html Msg
viewMetric metric =
    div [ class "metric" ]
        [ div [ class "label" ] [ text metric.label ]
        , div [ class "value" ] [ text (metric.formatter metric.value) ]
        , div [ class "graph" ]
            [ metric.graph metric.history
            , text (" (" ++ String.fromInt metric.minutes ++ "min)")
            ]
        ]


graphSize : Spark.Size
graphSize =
    { width = 180
    , height = 16
    , marginLR = 0
    , marginTB = 0
    }


areaStyle : Spark.Param a -> Spark.Param a
areaStyle =
    Spark.Style
        [ SvgAttrib.fill "rgba(50,100,255,0.3)"
        , SvgAttrib.stroke "rgba(50,100,255,1.0)"
        , SvgAttrib.strokeWidth "1.0"
        ]


barStyle : Spark.Param a -> Spark.Param a
barStyle =
    Spark.Style
        [ SvgAttrib.fill "rgba(50,200,50,0.7)"
        ]


zeroStyle : Spark.Param a -> Spark.Param a
zeroStyle =
    Spark.Style
        [ SvgAttrib.stroke "rgba(0,0,0,0.2)"
        , SvgAttrib.strokeWidth "1.0"
        ]


{-| Bar graph to be used with updateRemoteTotal metrics (change instead of absolute values).
-}
graphChange : Spark.DataSet -> Html a
graphChange data =
    let
        -- Used with Domain to stop sparkline forgetting about zero; continue scrolling graph.
        x =
            case List.head data of
                Nothing ->
                    0

                Just point ->
                    Tuple.first point
    in
    Spark.sparkline graphSize
        [ Spark.Bar 2.5 data |> barStyle
        , Spark.ZeroLine |> zeroStyle
        , Spark.Domain [ ( x, 0 ), ( x, 1 ) ]
        ]


{-| Zero based area graph, for charting absolute values relative to 0.
-}
graphZero : Spark.DataSet -> Html a
graphZero data =
    let
        -- Used with Domain to stop sparkline forgetting about zero; continue scrolling graph.
        x =
            case List.head data of
                Nothing ->
                    0

                Just point ->
                    Tuple.first point
    in
    Spark.sparkline graphSize
        [ Spark.Area data |> areaStyle
        , Spark.ZeroLine |> zeroStyle
        , Spark.Domain [ ( x, 0 ), ( x, 1 ) ]
        ]


framePanel : String -> String -> List (Html a) -> Html a
framePanel name icon html =
    let
        fontIcon cn =
            i [ class ("fas " ++ cn) ] []
    in
    div [ class "metric-panel" ]
        [ h2 []
            [ fontIcon icon
            , text " "
            , text name
            ]
        , div [ class "metrics" ] html
        ]



-- UTILS --


{-| Compute difference between each Int in numbers.
-}
changeList : List Int -> List Int
changeList numbers =
    let
        tail =
            List.tail numbers |> Maybe.withDefault []
    in
    List.map2 (-) tail numbers


{-| Pad the front of a list with 0s to make it at least 60 elements long.
-}
zeroPadList : List Int -> List Int
zeroPadList numbers =
    let
        needed =
            60 - List.length numbers
    in
    if needed > 0 then
        List.repeat needed 0 ++ numbers

    else
        numbers


{-| Format an Int with thousands separators.
-}
fmtInt : Int -> String
fmtInt n =
    let
        -- thousands recursively inserts thousands separators.
        thousands str =
            if String.length str <= 3 then
                str

            else
                thousands (String.slice 0 -3 str) ++ "," ++ String.right 3 str
    in
    thousands (String.fromInt n)
