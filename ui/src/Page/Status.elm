module Page.Status exposing (Model, Msg, init, load, subscriptions, update, view)

import Data.Metrics as Metrics exposing (Metrics)
import Data.Session as Session exposing (Session)
import Filesize
import Html exposing (..)
import Html.Attributes exposing (..)
import Http exposing (Error)
import HttpUtil
import Sparkline exposing (sparkline, Point, DataSet, Size)
import Svg.Attributes as SvgAttrib
import Time exposing (Time)


-- MODEL --


type alias Model =
    { metrics : Maybe Metrics
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
    , graph : DataSet -> Html Msg
    , history : DataSet
    , minutes : Int
    }


init : Model
init =
    { metrics = Nothing
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
    , smtpErrorsTotal = Metric "Messages Errors" 0 fmtInt graphChange initDataSet 60
    , smtpWarnsTotal = Metric "Messages Warns" 0 fmtInt graphChange initDataSet 60
    , retentionDeletesTotal = Metric "Retention Deletes" 0 fmtInt graphChange initDataSet 60
    , retainedCount = Metric "Stored Messages" 0 fmtInt graphZero initDataSet 60
    , retainedSize = Metric "Store Size" 0 Filesize.format graphZero initDataSet 60
    }


initDataSet : DataSet
initDataSet =
    List.range 0 59
        |> List.map (\x -> ( toFloat (x), 0 ))


load : Cmd Msg
load =
    getMetrics



-- SUBSCRIPTIONS --


subscriptions : Model -> Sub Msg
subscriptions model =
    Time.every (10 * Time.second) Tick



-- UPDATE --


type Msg
    = NewMetrics (Result Http.Error Metrics)
    | Tick Time


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    case msg of
        NewMetrics (Ok metrics) ->
            ( updateMetrics metrics model, Cmd.none, Session.none )

        NewMetrics (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )

        Tick time ->
            ( model, getMetrics, Session.ClearFlash )


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
            (Maybe.withDefault [] (List.tail metric.history))
                ++ [ ( x, (toFloat value) ) ]
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


getMetrics : Cmd Msg
getMetrics =
    Http.get "/debug/vars" Metrics.decoder
        |> Http.send NewMetrics



-- VIEW --


view : Session -> Model -> Html Msg
view session model =
    div [ id "page" ]
        [ h1 [] [ text "Status" ]
        , case model.metrics of
            Nothing ->
                div [] [ text "Loading metrics..." ]

            Just metrics ->
                div []
                    [ framePanel "General Metrics"
                        [ viewMetric model.sysMem
                        , viewMetric model.heapSize
                        , viewMetric model.heapUsed
                        , viewMetric model.heapObjects
                        , viewMetric model.goRoutines
                        , viewMetric model.webSockets
                        ]
                    , framePanel "SMTP Metrics"
                        [ viewMetric model.smtpConnOpen
                        , viewMetric model.smtpConnTotal
                        , viewMetric model.smtpReceivedTotal
                        , viewMetric model.smtpErrorsTotal
                        , viewMetric model.smtpWarnsTotal
                        ]
                    , framePanel "Storage Metrics"
                        [ viewMetric model.retentionDeletesTotal
                        , viewMetric model.retainedCount
                        , viewMetric model.retainedSize
                        ]
                    ]
        ]


viewMetric : Metric -> Html Msg
viewMetric metric =
    div [ class "metric" ]
        [ div [ class "label" ] [ text metric.label ]
        , div [ class "value" ] [ text (metric.formatter metric.value) ]
        , div [ class "graph" ]
            [ metric.graph metric.history
            , text ("(" ++ toString metric.minutes ++ "min)")
            ]
        ]


viewLiveMetric : String -> (Int -> String) -> Int -> Html a -> Html a
viewLiveMetric label formatter value graph =
    div [ class "metric" ]
        [ div [ class "label" ] [ text label ]
        , div [ class "value" ] [ text (formatter value) ]
        , div [ class "graph" ]
            [ graph
            , text "(10min)"
            ]
        ]


graphNull : Html a
graphNull =
    div [] []


graphSize : Size
graphSize =
    ( 180, 16, 0, 0 )


areaStyle : Sparkline.Param a -> Sparkline.Param a
areaStyle =
    Sparkline.Style
        [ SvgAttrib.fill "rgba(50,100,255,0.3)"
        , SvgAttrib.stroke "rgba(50,100,255,1.0)"
        , SvgAttrib.strokeWidth "1.0"
        ]


barStyle : Sparkline.Param a -> Sparkline.Param a
barStyle =
    Sparkline.Style
        [ SvgAttrib.fill "rgba(50,200,50,0.7)"
        ]


zeroStyle : Sparkline.Param a -> Sparkline.Param a
zeroStyle =
    Sparkline.Style
        [ SvgAttrib.stroke "rgba(0,0,0,0.2)"
        , SvgAttrib.strokeWidth "1.0"
        ]


{-| Bar graph to be used with updateRemoteTotal metrics (change instead of absolute values).
-}
graphChange : DataSet -> Html a
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
        sparkline graphSize
            [ Sparkline.Bar 2.5 data |> barStyle
            , Sparkline.ZeroLine |> zeroStyle
            , Sparkline.Domain [ ( x, 0 ), ( x, 1 ) ]
            ]


{-| Zero based area graph, for charting absolute values relative to 0.
-}
graphZero : DataSet -> Html a
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
        sparkline graphSize
            [ Sparkline.Area data |> areaStyle
            , Sparkline.ZeroLine |> zeroStyle
            , Sparkline.Domain [ ( x, 0 ), ( x, 1 ) ]
            ]


framePanel : String -> List (Html a) -> Html a
framePanel name html =
    div [ class "metric-panel" ]
        [ h2 [] [ text name ]
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
            (List.repeat needed 0) ++ numbers
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
                (thousands (String.slice 0 -3 str)) ++ "," ++ (String.right 3 str)
    in
        thousands (toString n)
