module Page.Monitor exposing (Model, Msg, init, subscriptions, update, view)

import Data.MessageHeader as MessageHeader exposing (MessageHeader)
import Data.Session as Session exposing (Session)
import DateFormat
    exposing
        ( amPmUppercase
        , dayOfMonthFixed
        , format
        , hourNumber
        , minuteFixed
        , monthNameAbbreviated
        )
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events as Events
import Json.Decode as D
import Ports
import Route
import Time exposing (Posix)



-- MODEL


type alias Model =
    { connected : Bool
    , messages : List MessageHeader
    }


init : ( Model, Cmd Msg )
init =
    ( Model False []
    , Cmd.batch
        [ Ports.windowTitle "Inbucket Monitor"
        , Ports.monitorCommand True
        ]
    )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    let
        monitorMessage =
            D.oneOf
                [ D.map Message MessageHeader.decoder
                , D.map Connected D.bool
                ]
                |> D.decodeValue
                |> Ports.monitorMessage
    in
    Sub.map MonitorResult monitorMessage



-- UPDATE


type Msg
    = MonitorResult (Result D.Error MonitorMessage)
    | OpenMessage MessageHeader


type MonitorMessage
    = Connected Bool
    | Message MessageHeader


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    case msg of
        MonitorResult (Ok (Connected status)) ->
            ( { model | connected = status }, Cmd.none, Session.none )

        MonitorResult (Ok (Message header)) ->
            ( { model | messages = header :: model.messages }, Cmd.none, Session.none )

        MonitorResult (Err err) ->
            ( model, Cmd.none, Session.SetFlash (D.errorToString err) )

        OpenMessage header ->
            ( model
            , Route.newUrl session.key (Route.Message header.mailbox header.id)
            , Session.none
            )



-- VIEW


view : Session -> Model -> Html Msg
view session model =
    div [ id "page" ]
        [ h1 [] [ text "Monitor" ]
        , p []
            [ text "Messages will be listed here shortly after delivery. "
            , em []
                [ text
                    (if model.connected then
                        "Connected."

                     else
                        "Disconnected!"
                    )
                ]
            ]
        , table [ id "monitor" ]
            [ thead []
                [ th [] [ text "Date" ]
                , th [ class "desktop" ] [ text "From" ]
                , th [] [ text "Mailbox" ]
                , th [] [ text "Subject" ]
                ]
            , tbody [] (List.map viewMessage model.messages)
            ]
        ]


viewMessage : MessageHeader -> Html Msg
viewMessage message =
    tr [ Events.onClick (OpenMessage message) ]
        [ td [] [ shortDate message.date ]
        , td [ class "desktop" ] [ text message.from ]
        , td [] [ text message.mailbox ]
        , td [] [ text message.subject ]
        ]


shortDate : Posix -> Html Msg
shortDate date =
    format
        [ dayOfMonthFixed
        , DateFormat.text "-"
        , monthNameAbbreviated
        , DateFormat.text " "
        , hourNumber
        , DateFormat.text ":"
        , minuteFixed
        , DateFormat.text " "
        , amPmUppercase
        ]
        Time.utc
        date
        |> text
