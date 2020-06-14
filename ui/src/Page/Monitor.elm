module Page.Monitor exposing (Model, Msg, init, update, view)

import Browser.Navigation as Nav
import Data.MessageHeader as MessageHeader exposing (MessageHeader)
import Data.Session as Session exposing (Session)
import DateFormat as DF
import Html
    exposing
        ( Attribute
        , Html
        , button
        , div
        , em
        , h1
        , node
        , span
        , table
        , tbody
        , td
        , text
        , th
        , thead
        , tr
        )
import Html.Attributes exposing (class, tabindex)
import Html.Events as Events
import Json.Decode as D
import Route
import Time exposing (Posix)



-- MODEL


type alias Model =
    { session : Session
    , connected : Bool
    , messages : List MessageHeader
    }


init : Session -> ( Model, Cmd Msg )
init session =
    ( Model session False [], Cmd.none )



-- UPDATE


type Msg
    = Connected Bool
    | MessageReceived D.Value
    | Clear
    | OpenMessage MessageHeader
    | MessageKeyPress MessageHeader Int


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        Connected True ->
            ( { model | connected = True, messages = [] }, Cmd.none )

        Connected False ->
            ( { model | connected = False }, Cmd.none )

        MessageReceived value ->
            case D.decodeValue (MessageHeader.decoder |> D.at [ "detail" ]) value of
                Ok header ->
                    ( { model | messages = header :: List.take 500 model.messages }
                    , Cmd.none
                    )

                Err err ->
                    let
                        flash =
                            { title = "Message decoding failed"
                            , table = [ ( "Error", D.errorToString err ) ]
                            }
                    in
                    ( { model | session = Session.showFlash flash model.session }
                    , Cmd.none
                    )

        Clear ->
            ( { model | messages = [] }, Cmd.none )

        OpenMessage header ->
            openMessage header model

        MessageKeyPress header keyCode ->
            case keyCode of
                13 ->
                    openMessage header model

                _ ->
                    ( model, Cmd.none )


openMessage : MessageHeader -> Model -> ( Model, Cmd Msg )
openMessage header model =
    ( model
    , Route.Message header.mailbox header.id
        |> model.session.router.toPath
        |> Nav.replaceUrl model.session.key
    )



-- VIEW


view : Model -> { title : String, modal : Maybe (Html msg), content : List (Html Msg) }
view model =
    { title = "Inbucket Monitor"
    , modal = Nothing
    , content =
        [ h1 [] [ text "Monitor" ]
        , div [ class "monitor-header" ]
            [ span [ class "monitor-description" ]
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
            , span [ class "button-bar monitor-buttons" ]
                [ button [ Events.onClick Clear ] [ text "Clear" ]
                ]
            ]
        , node "monitor-messages"
            [ Events.on "connected" (D.map Connected <| D.at [ "detail" ] <| D.bool)
            , Events.on "message" (D.map MessageReceived D.value)
            ]
            []
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
    tr
        [ tabindex 0
        , Events.onClick (OpenMessage message)
        , onKeyUp (MessageKeyPress message)
        ]
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
        , DF.text ":"
        , DF.secondFixed
        , DF.text " "
        , DF.amPmUppercase
        ]
        zone
        date
        |> text


onKeyUp : (Int -> msg) -> Attribute msg
onKeyUp tagger =
    Events.on "keyup" (D.map tagger Events.keyCode)
