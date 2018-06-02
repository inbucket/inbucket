module Page.Mailbox exposing (Model, Msg, init, load, update, view)

import Data.Message as Message exposing (Message)
import Data.MessageHeader as MessageHeader exposing (MessageHeader)
import Data.Session as Session exposing (Session)
import Json.Decode as Decode exposing (Decoder)
import Html exposing (..)
import Html.Attributes exposing (class, classList, href, id, placeholder, target)
import Html.Events exposing (..)
import Http exposing (Error)
import HttpUtil
import Ports
import Route exposing (Route)


inbucketBase : String
inbucketBase =
    ""



-- MODEL --


type alias Model =
    { name : String
    , selected : Maybe String
    , headers : List MessageHeader
    , message : Maybe Message
    }


init : String -> Maybe String -> Model
init name id =
    Model name id [] Nothing


load : String -> Cmd Msg
load name =
    Cmd.batch
        [ Ports.windowTitle (name ++ " - Inbucket")
        , getMailbox name
        ]



-- UPDATE --


type Msg
    = ClickMessage String
    | ViewMessage String
    | DeleteMessage Message
    | DeleteMessageResult (Result Http.Error ())
    | NewMailbox (Result Http.Error (List MessageHeader))
    | NewMessage (Result Http.Error Message)


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    case msg of
        ClickMessage id ->
            ( { model | selected = Just id }
            , Cmd.batch
                [ Route.newUrl (Route.Message model.name id)
                , getMessage model.name id
                ]
            , Session.DisableRouting
            )

        ViewMessage id ->
            ( { model | selected = Just id }
            , getMessage model.name id
            , Session.AddRecent model.name
            )

        DeleteMessage msg ->
            deleteMessage model msg

        DeleteMessageResult (Ok _) ->
            ( model, Cmd.none, Session.none )

        DeleteMessageResult (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )

        NewMailbox (Ok headers) ->
            let
                newModel =
                    { model | headers = headers }
            in
                case model.selected of
                    Nothing ->
                        ( newModel, Cmd.none, Session.AddRecent model.name )

                    Just id ->
                        -- Recurse to select message id.
                        update session (ViewMessage id) newModel

        NewMailbox (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )

        NewMessage (Ok msg) ->
            ( { model | message = Just msg }, Cmd.none, Session.none )

        NewMessage (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )


getMailbox : String -> Cmd Msg
getMailbox name =
    let
        url =
            inbucketBase ++ "/api/v1/mailbox/" ++ name
    in
        Http.get url (Decode.list MessageHeader.decoder)
            |> Http.send NewMailbox


deleteMessage : Model -> Message -> ( Model, Cmd Msg, Session.Msg )
deleteMessage model msg =
    let
        url =
            inbucketBase ++ "/api/v1/mailbox/" ++ msg.mailbox ++ "/" ++ msg.id

        cmd =
            HttpUtil.delete url
                |> Http.send DeleteMessageResult
    in
        ( { model
            | message = Nothing
            , selected = Nothing
            , headers = List.filter (\x -> x.id /= msg.id) model.headers
          }
        , cmd
        , Session.none
        )


getMessage : String -> String -> Cmd Msg
getMessage mailbox id =
    let
        url =
            inbucketBase ++ "/api/v1/mailbox/" ++ mailbox ++ "/" ++ id
    in
        Http.get url Message.decoder
            |> Http.send NewMessage



-- VIEW --


view : Session -> Model -> Html Msg
view session model =
    div [ id "page", class "mailbox" ]
        [ aside [ id "message-list" ] [ viewMailbox model ]
        , main_ [ id "message" ] [ viewMessage model ]
        ]


viewMailbox : Model -> Html Msg
viewMailbox model =
    div [] (List.map (viewHeader model) (List.reverse model.headers))


viewHeader : Model -> MessageHeader -> Html Msg
viewHeader mailbox msg =
    div
        [ classList
            [ ( "message-list-entry", True )
            , ( "selected", mailbox.selected == Just msg.id )
            , ( "unseen", not msg.seen )
            ]
        , onClick (ClickMessage msg.id)
        ]
        [ div [ class "subject" ] [ text msg.subject ]
        , div [ class "from" ] [ text msg.from ]
        , div [ class "date" ] [ text msg.date ]
        ]


viewMessage : Model -> Html Msg
viewMessage model =
    case model.message of
        Just message ->
            div []
                [ div [ class "button-bar" ]
                    [ button [ class "danger", onClick (DeleteMessage message) ] [ text "Delete" ]
                    , a
                        [ href
                            (inbucketBase
                                ++ "/mailbox/"
                                ++ message.mailbox
                                ++ "/"
                                ++ message.id
                                ++ "/source"
                            )
                        , target "_blank"
                        ]
                        [ button [] [ text "Source" ] ]
                    ]
                , dl [ id "message-header" ]
                    [ dt [] [ text "From:" ]
                    , dd [] [ text message.from ]
                    , dt [] [ text "To:" ]
                    , dd [] (List.map text message.to)
                    , dt [] [ text "Date:" ]
                    , dd [] [ text message.date ]
                    , dt [] [ text "Subject:" ]
                    , dd [] [ text message.subject ]
                    ]
                , article [] [ text message.body.text ]
                ]

        Nothing ->
            text ""
