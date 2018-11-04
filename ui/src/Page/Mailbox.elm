module Page.Mailbox exposing (Model, Msg, init, load, subscriptions, update, view)

import Data.Message as Message exposing (Message)
import Data.MessageHeader as MessageHeader exposing (MessageHeader)
import Data.Session as Session exposing (Session)
import Json.Decode as Decode exposing (Decoder)
import Html exposing (..)
import Html.Attributes exposing (class, classList, downloadAs, href, id, property, target)
import Html.Events exposing (..)
import Http exposing (Error)
import HttpUtil
import Json.Encode as Encode
import Ports
import Route
import Task
import Time exposing (Time)


-- MODEL


type Body
    = TextBody
    | SafeHtmlBody


type alias Model =
    { name : String
    , selected : Maybe String
    , headers : List MessageHeader
    , visible :
        Maybe
            { message : Message
            , markSeenAt : Maybe Time
            }
    , bodyMode : Body
    }


init : String -> Maybe String -> Model
init name id =
    Model name id [] Nothing SafeHtmlBody


load : String -> Cmd Msg
load name =
    Cmd.batch
        [ Ports.windowTitle (name ++ " - Inbucket")
        , getMailbox name
        ]



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    case model.visible of
        Just { message, markSeenAt } ->
            case markSeenAt of
                Just time ->
                    Time.every (250 * Time.millisecond) Tick

                Nothing ->
                    Sub.none

        Nothing ->
            Sub.none



-- UPDATE


type Msg
    = ClickMessage String
    | ViewMessage String
    | DeleteMessage Message
    | DeleteMessageResult (Result Http.Error ())
    | MailboxResult (Result Http.Error (List MessageHeader))
    | MarkSeenResult (Result Http.Error ())
    | MessageResult (Result Http.Error Message)
    | MessageBody Body
    | OpenedTime Time
    | Tick Time


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

        MailboxResult (Ok headers) ->
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

        MailboxResult (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )

        MarkSeenResult (Ok _) ->
            ( model, Cmd.none, Session.none )

        MarkSeenResult (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )

        MessageResult (Ok msg) ->
            let
                bodyMode =
                    if msg.html == "" then
                        TextBody
                    else
                        model.bodyMode
            in
                ( { model
                    | visible = Just { message = msg, markSeenAt = Nothing }
                    , bodyMode = bodyMode
                  }
                , Task.perform OpenedTime Time.now
                , Session.none
                )

        MessageResult (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )

        MessageBody bodyMode ->
            ( { model | bodyMode = bodyMode }, Cmd.none, Session.none )

        OpenedTime time ->
            case model.visible of
                Just visible ->
                    if visible.message.seen then
                        ( model, Cmd.none, Session.none )
                    else
                        -- Set delay to report message as seen to backend.
                        ( { model
                            | visible =
                                Just
                                    { visible
                                        | markSeenAt = Just (time + (1.5 * Time.second))
                                    }
                          }
                        , Cmd.none
                        , Session.none
                        )

                Nothing ->
                    ( model, Cmd.none, Session.none )

        Tick now ->
            case model.visible of
                Just { message, markSeenAt } ->
                    case markSeenAt of
                        Just deadline ->
                            if now >= deadline then
                                markMessageSeen model message
                            else
                                ( model, Cmd.none, Session.none )

                        Nothing ->
                            ( model, Cmd.none, Session.none )

                Nothing ->
                    ( model, Cmd.none, Session.none )


getMailbox : String -> Cmd Msg
getMailbox name =
    let
        url =
            "/api/v1/mailbox/" ++ name
    in
        Http.get url (Decode.list MessageHeader.decoder)
            |> Http.send MailboxResult


deleteMessage : Model -> Message -> ( Model, Cmd Msg, Session.Msg )
deleteMessage model msg =
    let
        url =
            "/api/v1/mailbox/" ++ msg.mailbox ++ "/" ++ msg.id

        cmd =
            HttpUtil.delete url
                |> Http.send DeleteMessageResult
    in
        ( { model
            | visible = Nothing
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
            "/serve/m/" ++ mailbox ++ "/" ++ id
    in
        Http.get url Message.decoder
            |> Http.send MessageResult


markMessageSeen : Model -> Message -> ( Model, Cmd Msg, Session.Msg )
markMessageSeen model message =
    case model.visible of
        Just visible ->
            let
                updateSeen header =
                    if header.id == message.id then
                        { header | seen = True }
                    else
                        header

                url =
                    "/api/v1/mailbox/" ++ message.mailbox ++ "/" ++ message.id

                command =
                    -- The URL tells the API what message to update, so we only need to indicate the
                    -- desired change in the body.
                    Encode.object [ ( "seen", Encode.bool True ) ]
                        |> Http.jsonBody
                        |> HttpUtil.patch url
                        |> Http.send MarkSeenResult
            in
                ( { model
                    | visible = Just { visible | markSeenAt = Nothing }
                    , headers = List.map updateSeen model.headers
                  }
                , command
                , Session.None
                )

        Nothing ->
            ( model, Cmd.none, Session.none )



-- VIEW


view : Session -> Model -> Html Msg
view session model =
    div [ id "page", class "mailbox" ]
        [ aside [ id "message-list" ] [ messageList model ]
        , main_
            [ id "message" ]
            [ case model.visible of
                Just { message } ->
                    viewMessage message model.bodyMode

                Nothing ->
                    text
                        ("Select a message on the left,"
                            ++ " or enter a different username into the box on upper right."
                        )
            ]
        ]


messageList : Model -> Html Msg
messageList model =
    div [] (List.map (messageChip model.selected) (List.reverse model.headers))


messageChip : Maybe String -> MessageHeader -> Html Msg
messageChip selected msg =
    div
        [ classList
            [ ( "message-list-entry", True )
            , ( "selected", selected == Just msg.id )
            , ( "unseen", not msg.seen )
            ]
        , onClick (ClickMessage msg.id)
        ]
        [ div [ class "subject" ] [ text msg.subject ]
        , div [ class "from" ] [ text msg.from ]
        , div [ class "date" ] [ text msg.date ]
        ]


viewMessage : Message -> Body -> Html Msg
viewMessage message bodyMode =
    let
        sourceUrl message =
            "/serve/m/" ++ message.mailbox ++ "/" ++ message.id ++ "/source"
    in
        div []
            [ div [ class "button-bar" ]
                [ button [ class "danger", onClick (DeleteMessage message) ] [ text "Delete" ]
                , a
                    [ href (sourceUrl message), target "_blank" ]
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
            , messageBody message bodyMode
            , attachments message
            ]


messageBody : Message -> Body -> Html Msg
messageBody message bodyMode =
    let
        bodyModeTab mode label =
            a
                [ classList [ ( "active", bodyMode == mode ) ]
                , onClick (MessageBody mode)
                , href "javacript:void(0)"
                ]
                [ text label ]

        safeHtml =
            bodyModeTab SafeHtmlBody "Safe HTML"

        plainText =
            bodyModeTab TextBody "Plain Text"

        tabs =
            if message.html == "" then
                [ plainText ]
            else
                [ safeHtml, plainText ]
    in
        div [ class "tab-panel" ]
            [ nav [ class "tab-bar" ] tabs
            , article [ class "message-body" ]
                [ case bodyMode of
                    SafeHtmlBody ->
                        div [ property "innerHTML" (Encode.string message.html) ] []

                    TextBody ->
                        div [ property "innerHTML" (Encode.string message.text) ] []
                ]
            ]


attachments : Message -> Html Msg
attachments message =
    let
        baseUrl =
            "/serve/m/attach/" ++ message.mailbox ++ "/" ++ message.id ++ "/"
    in
        if List.isEmpty message.attachments then
            div [] []
        else
            table [ class "attachments well" ] (List.map (attachmentRow baseUrl) message.attachments)


attachmentRow : String -> Message.Attachment -> Html Msg
attachmentRow baseUrl attach =
    let
        url =
            baseUrl ++ attach.id ++ "/" ++ attach.fileName
    in
        tr []
            [ td []
                [ a [ href url, target "_blank" ] [ text attach.fileName ]
                , text (" (" ++ attach.contentType ++ ") ")
                ]
            , td [] [ a [ href url, downloadAs attach.fileName, class "button" ] [ text "Download" ] ]
            ]
