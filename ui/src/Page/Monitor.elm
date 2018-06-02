module Page.Monitor exposing (Model, Msg, init, subscriptions, update, view)

import Data.MessageHeader as MessageHeader exposing (MessageHeader)
import Data.Session as Session exposing (Session)
import Json.Decode exposing (decodeString)
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events as Events
import Route
import WebSocket


-- MODEL --


type alias Model =
    { messages : List MessageHeader }


init : Model
init =
    { messages = [] }



-- SUBSCRIPTIONS --


subscriptions : Model -> Sub Msg
subscriptions model =
    WebSocket.listen "ws://192.168.1.10:3000/api/v1/monitor/messages"
        (decodeString MessageHeader.decoder >> NewMessage)



-- UPDATE --


type Msg
    = NewMessage (Result String MessageHeader)
    | OpenMessage MessageHeader


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    case msg of
        NewMessage (Ok msg) ->
            ( { model | messages = msg :: model.messages }, Cmd.none, Session.none )

        NewMessage (Err err) ->
            ( model, Cmd.none, Session.SetFlash err )

        OpenMessage msg ->
            ( model
            , Route.newUrl (Route.Message msg.mailbox msg.id)
            , Session.none
            )



-- VIEW --


view : Session -> Model -> Html Msg
view session model =
    div [ id "page" ]
        [ h1 [] [ text "Monitor" ]
        , p [] [ text "Messages will be listed here shortly after delivery." ]
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
        [ td [] [ text message.date ]
        , td [ class "desktop" ] [ text message.from ]
        , td [] [ text message.mailbox ]
        , td [] [ text message.subject ]
        ]
