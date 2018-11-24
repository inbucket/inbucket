module Page.Monitor exposing (Model, Msg, init, subscriptions, update, view)

import Data.MessageHeader as MessageHeader exposing (MessageHeader)
import Data.Session as Session exposing (Session)
import DateFormat as DF
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


type MonitorMessage
    = Connected Bool
    | Message MessageHeader


init : ( Model, Cmd Msg, Session.Msg )
init =
    ( Model False [], Ports.monitorCommand True, Session.none )



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
    Sub.map MessageReceived monitorMessage



-- UPDATE


type Msg
    = MessageReceived (Result D.Error MonitorMessage)
    | OpenMessage MessageHeader


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    case msg of
        MessageReceived (Ok (Connected status)) ->
            ( { model | connected = status }, Cmd.none, Session.none )

        MessageReceived (Ok (Message header)) ->
            ( { model | messages = header :: model.messages }, Cmd.none, Session.none )

        MessageReceived (Err err) ->
            ( model, Cmd.none, Session.SetFlash (D.errorToString err) )

        OpenMessage header ->
            ( model
            , Route.pushUrl session.key (Route.Message header.mailbox header.id)
            , Session.none
            )



-- VIEW


view : Session -> Model -> { title : String, content : Html Msg }
view session model =
    { title = "Inbucket Monitor"
    , content =
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
                , tbody [] (List.map (viewMessage session.zone) model.messages)
                ]
            ]
    }


viewMessage : Time.Zone -> MessageHeader -> Html Msg
viewMessage zone message =
    tr [ Events.onClick (OpenMessage message) ]
        [ td [] [ shortDate zone message.date ]
        , td [ class "desktop" ] [ text message.from ]
        , td [] [ text message.mailbox ]
        , td [] [ text message.subject ]
        ]


shortDate : Time.Zone -> Posix -> Html Msg
shortDate zone date =
    DF.format
        [ DF.dayOfMonthFixed
        , DF.text "-"
        , DF.monthNameAbbreviated
        , DF.text " "
        , DF.hourNumber
        , DF.text ":"
        , DF.minuteFixed
        , DF.text " "
        , DF.amPmUppercase
        ]
        zone
        date
        |> text
