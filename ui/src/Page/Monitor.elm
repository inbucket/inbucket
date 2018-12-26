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
    { session : Session
    , connected : Bool
    , messages : List MessageHeader
    }


type MonitorMessage
    = Connected Bool
    | Message MessageHeader


init : Session -> ( Model, Cmd Msg )
init session =
    ( Model session False []
    , Ports.monitorCommand True
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
    Sub.map MessageReceived monitorMessage



-- UPDATE


type Msg
    = MessageReceived (Result D.Error MonitorMessage)
    | OpenMessage MessageHeader


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        MessageReceived (Ok (Connected status)) ->
            ( { model | connected = status }, Cmd.none )

        MessageReceived (Ok (Message header)) ->
            ( { model | messages = header :: model.messages }, Cmd.none )

        MessageReceived (Err err) ->
            let
                flash =
                    { title = "Decoding failed"
                    , table = [ ( "Error", D.errorToString err ) ]
                    }
            in
            ( { model | session = Session.showFlash flash model.session }
            , Cmd.none
            )

        OpenMessage header ->
            ( model
            , Route.pushUrl model.session.key (Route.Message header.mailbox header.id)
            )



-- VIEW


view : Model -> { title : String, modal : Maybe (Html msg), content : List (Html Msg) }
view model =
    { title = "Inbucket Monitor"
    , modal = Nothing
    , content =
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
        , table [ class "monitor" ]
            [ thead []
                [ th [] [ text "Date" ]
                , th [ class "desktop" ] [ text "From" ]
                , th [] [ text "Mailbox" ]
                , th [] [ text "Subject" ]
                ]
            , tbody [] (List.map (viewMessage model.session.zone) model.messages)
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
