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


type State
    = LoadingList (Maybe MessageID)
    | ShowingList (List MessageHeader) (Maybe MessageID)
    | LoadingMessage (List MessageHeader) MessageID
    | ShowingMessage (List MessageHeader) VisibleMessage
    | Transitioning (List MessageHeader) VisibleMessage MessageID


type alias MessageID =
    String


type alias VisibleMessage =
    { message : Message
    , markSeenAt : Maybe Time
    }


type alias Model =
    { mailboxName : String
    , state : State
    , bodyMode : Body
    }


init : String -> Maybe MessageID -> Model
init mailboxName selection =
    Model mailboxName (LoadingList selection) SafeHtmlBody


load : String -> Cmd Msg
load mailboxName =
    Cmd.batch
        [ Ports.windowTitle (mailboxName ++ " - Inbucket")
        , getList mailboxName
        ]



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    case model.state of
        ShowingMessage _ { message } ->
            if message.seen then
                Sub.none
            else
                Time.every (250 * Time.millisecond) Tick

        _ ->
            Sub.none



-- UPDATE


type Msg
    = ClickMessage MessageID
    | ViewMessage MessageID
    | DeleteMessage Message
    | DeleteMessageResult (Result Http.Error ())
    | ListResult (Result Http.Error (List MessageHeader))
    | MarkSeenResult (Result Http.Error ())
    | MessageResult (Result Http.Error Message)
    | MessageBody Body
    | OpenedTime Time
    | Tick Time


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    case msg of
        ClickMessage id ->
            ( updateSelected model id
            , Cmd.batch
                [ Route.newUrl (Route.Message model.mailboxName id)
                , getMessage model.mailboxName id
                ]
            , Session.DisableRouting
            )

        ViewMessage id ->
            ( updateSelected model id
            , getMessage model.mailboxName id
            , Session.AddRecent model.mailboxName
            )

        DeleteMessage msg ->
            deleteMessage model msg

        DeleteMessageResult (Ok _) ->
            ( model, Cmd.none, Session.none )

        DeleteMessageResult (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )

        ListResult (Ok headers) ->
            case model.state of
                LoadingList selection ->
                    let
                        newModel =
                            { model | state = ShowingList headers selection }
                    in
                        case selection of
                            Just id ->
                                -- Recurse to select message id.
                                update session (ViewMessage id) newModel

                            Nothing ->
                                ( newModel, Cmd.none, Session.AddRecent model.mailboxName )

                _ ->
                    ( model, Cmd.none, Session.none )

        ListResult (Err err) ->
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

                updateMessage list message =
                    ( { model
                        | state = ShowingMessage list { message = message, markSeenAt = Nothing }
                        , bodyMode = bodyMode
                      }
                    , Task.perform OpenedTime Time.now
                    , Session.none
                    )
            in
                case model.state of
                    LoadingList _ ->
                        ( model, Cmd.none, Session.none )

                    ShowingList list _ ->
                        updateMessage list msg

                    LoadingMessage list _ ->
                        updateMessage list msg

                    ShowingMessage list _ ->
                        updateMessage list msg

                    Transitioning list _ _ ->
                        updateMessage list msg

        MessageResult (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )

        MessageBody bodyMode ->
            ( { model | bodyMode = bodyMode }, Cmd.none, Session.none )

        OpenedTime time ->
            case model.state of
                ShowingMessage list visible ->
                    if visible.message.seen then
                        ( model, Cmd.none, Session.none )
                    else
                        -- Set delay to report message as seen to backend.
                        ( { model
                            | state =
                                ShowingMessage list
                                    { visible
                                        | markSeenAt = Just (time + (1.5 * Time.second))
                                    }
                          }
                        , Cmd.none
                        , Session.none
                        )

                _ ->
                    ( model, Cmd.none, Session.none )

        Tick now ->
            case model.state of
                ShowingMessage _ { message, markSeenAt } ->
                    case markSeenAt of
                        Just deadline ->
                            if now >= deadline then
                                markMessageSeen model message
                            else
                                ( model, Cmd.none, Session.none )

                        Nothing ->
                            ( model, Cmd.none, Session.none )

                _ ->
                    ( model, Cmd.none, Session.none )


updateSelected : Model -> MessageID -> Model
updateSelected model id =
    case model.state of
        ShowingList list _ ->
            { model | state = LoadingMessage list id }

        ShowingMessage list visible ->
            -- Use Transitioning state to prevent message flicker.
            { model | state = Transitioning list visible id }

        Transitioning list visible _ ->
            { model | state = Transitioning list visible id }

        _ ->
            model


getList : String -> Cmd Msg
getList mailboxName =
    let
        url =
            "/api/v1/mailbox/" ++ mailboxName
    in
        Http.get url (Decode.list MessageHeader.decoder)
            |> Http.send ListResult


deleteMessage : Model -> Message -> ( Model, Cmd Msg, Session.Msg )
deleteMessage model msg =
    let
        url =
            "/api/v1/mailbox/" ++ msg.mailbox ++ "/" ++ msg.id

        cmd =
            HttpUtil.delete url
                |> Http.send DeleteMessageResult
    in
        case model.state of
            ShowingMessage list _ ->
                ( { model | state = ShowingList (List.filter (\x -> x.id /= msg.id) list) Nothing }
                , cmd
                , Session.none
                )

            _ ->
                ( model, cmd, Session.none )


getMessage : String -> MessageID -> Cmd Msg
getMessage mailboxName id =
    let
        url =
            "/serve/m/" ++ mailboxName ++ "/" ++ id
    in
        Http.get url Message.decoder
            |> Http.send MessageResult


markMessageSeen : Model -> Message -> ( Model, Cmd Msg, Session.Msg )
markMessageSeen model message =
    case model.state of
        ShowingMessage list visible ->
            let
                message =
                    visible.message

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
                    | state =
                        ShowingMessage (List.map updateSeen list)
                            { visible
                                | message = { message | seen = True }
                                , markSeenAt = Nothing
                            }
                  }
                , command
                , Session.None
                )

        _ ->
            ( model, Cmd.none, Session.none )



-- VIEW


view : Session -> Model -> Html Msg
view session model =
    div [ id "page", class "mailbox" ]
        [ aside [ id "message-list" ]
            [ case model.state of
                LoadingList _ ->
                    messageList [] Nothing

                ShowingList list selection ->
                    messageList list selection

                LoadingMessage list selection ->
                    messageList list (Just selection)

                ShowingMessage list visible ->
                    messageList list (Just visible.message.id)

                Transitioning list _ selection ->
                    messageList list (Just selection)
            ]
        , main_
            [ id "message" ]
            [ case model.state of
                ShowingList _ _ ->
                    text
                        ("Select a message on the left,"
                            ++ " or enter a different username into the box on upper right."
                        )

                ShowingMessage _ { message } ->
                    viewMessage message model.bodyMode

                Transitioning _ { message } _ ->
                    viewMessage message model.bodyMode

                _ ->
                    text ""
            ]
        ]


messageList : List MessageHeader -> Maybe MessageID -> Html Msg
messageList list selected =
    div [] (List.map (messageChip selected) (List.reverse list))


messageChip : Maybe MessageID -> MessageHeader -> Html Msg
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
