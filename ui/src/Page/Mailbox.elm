module Page.Mailbox exposing (Model, Msg, init, load, subscriptions, update, view)

import Api
import Data.Message as Message exposing (Message)
import Data.MessageHeader as MessageHeader exposing (MessageHeader)
import Data.Session as Session exposing (Session)
import DateFormat as DF
import DateFormat.Relative as Relative
import Html exposing (..)
import Html.Attributes
    exposing
        ( class
        , classList
        , download
        , href
        , id
        , placeholder
        , property
        , target
        , type_
        , value
        )
import Html.Events exposing (..)
import Http exposing (Error)
import HttpUtil
import Json.Decode as Decode exposing (Decoder)
import Json.Encode as Encode
import Ports
import Route
import Task
import Time exposing (Posix)



-- MODEL


type Body
    = TextBody
    | SafeHtmlBody


type State
    = LoadingList (Maybe MessageID)
    | ShowingList MessageList MessageState


type MessageState
    = NoMessage
    | LoadingMessage
    | ShowingMessage VisibleMessage
    | Transitioning VisibleMessage


type alias MessageID =
    String


type alias MessageList =
    { headers : List MessageHeader
    , selected : Maybe MessageID
    , searchFilter : String
    }


type alias VisibleMessage =
    { message : Message
    , markSeenAt : Maybe Int
    }


type alias Model =
    { mailboxName : String
    , state : State
    , bodyMode : Body
    , searchInput : String
    , promptPurge : Bool
    , now : Posix
    }


init : String -> Maybe MessageID -> ( Model, Cmd Msg, Session.Msg )
init mailboxName selection =
    ( Model mailboxName (LoadingList selection) SafeHtmlBody "" False (Time.millisToPosix 0)
    , load mailboxName
    , Session.none
    )


load : String -> Cmd Msg
load mailboxName =
    Cmd.batch
        [ Task.perform Tick Time.now
        , Api.getHeaderList ListLoaded mailboxName
        ]



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    let
        subSeen =
            case model.state of
                ShowingList _ (ShowingMessage { message }) ->
                    if message.seen then
                        Sub.none

                    else
                        Time.every 250 MarkSeenTick

                _ ->
                    Sub.none
    in
    Sub.batch
        [ Time.every (30 * 1000) Tick
        , subSeen
        ]



-- UPDATE


type Msg
    = ListLoaded (Result HttpUtil.Error (List MessageHeader))
    | ClickMessage MessageID
    | OpenMessage MessageID
    | MessageLoaded (Result HttpUtil.Error Message)
    | MessageBody Body
    | OpenedTime Posix
    | MarkSeenTick Posix
    | MarkedSeen (Result HttpUtil.Error ())
    | DeleteMessage Message
    | DeletedMessage (Result HttpUtil.Error ())
    | PurgeMailboxPrompt
    | PurgeMailboxCanceled
    | PurgeMailboxConfirmed
    | PurgedMailbox (Result HttpUtil.Error ())
    | OnSearchInput String
    | Tick Posix


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    case msg of
        ClickMessage id ->
            ( updateSelected model id
            , Cmd.batch
                [ -- Update browser location.
                  Route.replaceUrl session.key (Route.Message model.mailboxName id)
                , Api.getMessage MessageLoaded model.mailboxName id
                ]
            , Session.DisableRouting
            )

        OpenMessage id ->
            updateOpenMessage session model id

        DeleteMessage message ->
            updateDeleteMessage session model message

        DeletedMessage (Ok _) ->
            ( model, Cmd.none, Session.none )

        DeletedMessage (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorFlash err) )

        ListLoaded (Ok headers) ->
            case model.state of
                LoadingList selection ->
                    let
                        newModel =
                            { model
                                | state = ShowingList (MessageList headers Nothing "") NoMessage
                            }
                    in
                    case selection of
                        Just id ->
                            updateOpenMessage session newModel id

                        Nothing ->
                            ( newModel, Cmd.none, Session.AddRecent model.mailboxName )

                _ ->
                    ( model, Cmd.none, Session.none )

        ListLoaded (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorFlash err) )

        MarkedSeen (Ok _) ->
            ( model, Cmd.none, Session.none )

        MarkedSeen (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorFlash err) )

        MessageLoaded (Ok message) ->
            updateMessageResult model message

        MessageLoaded (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorFlash err) )

        MessageBody bodyMode ->
            ( { model | bodyMode = bodyMode }, Cmd.none, Session.none )

        OnSearchInput searchInput ->
            updateSearchInput model searchInput

        OpenedTime time ->
            case model.state of
                ShowingList list (ShowingMessage visible) ->
                    if visible.message.seen then
                        ( model, Cmd.none, Session.none )

                    else
                        -- Set 1500ms delay before reporting message as seen to backend.
                        let
                            markSeenAt =
                                Time.posixToMillis time + 1500
                        in
                        ( { model
                            | state =
                                ShowingList list
                                    (ShowingMessage
                                        { visible
                                            | markSeenAt = Just markSeenAt
                                        }
                                    )
                          }
                        , Cmd.none
                        , Session.none
                        )

                _ ->
                    ( model, Cmd.none, Session.none )

        PurgeMailboxPrompt ->
            ( { model | promptPurge = True }, Cmd.none, Session.none )

        PurgeMailboxCanceled ->
            ( { model | promptPurge = False }, Cmd.none, Session.none )

        PurgeMailboxConfirmed ->
            updatePurge session model

        PurgedMailbox (Ok _) ->
            ( model, Cmd.none, Session.none )

        PurgedMailbox (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorFlash err) )

        MarkSeenTick now ->
            case model.state of
                ShowingList _ (ShowingMessage { message, markSeenAt }) ->
                    case markSeenAt of
                        Just deadline ->
                            if Time.posixToMillis now >= deadline then
                                updateMarkMessageSeen model message

                            else
                                ( model, Cmd.none, Session.none )

                        Nothing ->
                            ( model, Cmd.none, Session.none )

                _ ->
                    ( model, Cmd.none, Session.none )

        Tick now ->
            ( { model | now = now }, Cmd.none, Session.none )


{-| Replace the currently displayed message.
-}
updateMessageResult : Model -> Message -> ( Model, Cmd Msg, Session.Msg )
updateMessageResult model message =
    let
        bodyMode =
            if message.html == "" then
                TextBody

            else
                model.bodyMode
    in
    case model.state of
        LoadingList _ ->
            ( model, Cmd.none, Session.none )

        ShowingList list _ ->
            ( { model
                | state =
                    ShowingList
                        { list | selected = Just message.id }
                        (ShowingMessage (VisibleMessage message Nothing))
                , bodyMode = bodyMode
              }
            , Task.perform OpenedTime Time.now
            , Session.none
            )


updatePurge : Session -> Model -> ( Model, Cmd Msg, Session.Msg )
updatePurge session model =
    let
        cmd =
            Cmd.batch
                [ Route.replaceUrl session.key (Route.Mailbox model.mailboxName)
                , Api.purgeMailbox PurgedMailbox model.mailboxName
                ]
    in
    case model.state of
        ShowingList list _ ->
            ( { model
                | promptPurge = False
                , state = ShowingList (MessageList [] Nothing "") NoMessage
              }
            , cmd
            , Session.DisableRouting
            )

        _ ->
            ( model, cmd, Session.none )


updateSearchInput : Model -> String -> ( Model, Cmd Msg, Session.Msg )
updateSearchInput model searchInput =
    let
        searchFilter =
            if String.length searchInput > 1 then
                String.toLower searchInput

            else
                ""
    in
    case model.state of
        LoadingList _ ->
            ( model, Cmd.none, Session.none )

        ShowingList list messageState ->
            ( { model
                | searchInput = searchInput
                , state = ShowingList { list | searchFilter = searchFilter } messageState
              }
            , Cmd.none
            , Session.none
            )


{-| Set the selected message in our model.
-}
updateSelected : Model -> MessageID -> Model
updateSelected model id =
    case model.state of
        LoadingList _ ->
            model

        ShowingList list messageState ->
            let
                newList =
                    { list | selected = Just id }
            in
            case messageState of
                NoMessage ->
                    { model | state = ShowingList newList LoadingMessage }

                LoadingMessage ->
                    { model | state = ShowingList newList LoadingMessage }

                ShowingMessage visible ->
                    -- Use Transitioning state to prevent blank message flicker.
                    { model | state = ShowingList newList (Transitioning visible) }

                Transitioning visible ->
                    { model | state = ShowingList newList (Transitioning visible) }


updateDeleteMessage : Session -> Model -> Message -> ( Model, Cmd Msg, Session.Msg )
updateDeleteMessage session model message =
    let
        filter f messageList =
            { messageList | headers = List.filter f messageList.headers }
    in
    case model.state of
        ShowingList list _ ->
            ( { model
                | state =
                    ShowingList (filter (\x -> x.id /= message.id) list) NoMessage
              }
            , Cmd.batch
                [ Api.deleteMessage DeletedMessage message.mailbox message.id
                , Route.replaceUrl session.key (Route.Mailbox model.mailboxName)
                ]
            , Session.DisableRouting
            )

        _ ->
            ( model, Cmd.none, Session.none )


updateMarkMessageSeen : Model -> Message -> ( Model, Cmd Msg, Session.Msg )
updateMarkMessageSeen model message =
    case model.state of
        ShowingList list (ShowingMessage visible) ->
            let
                updateSeen header =
                    if header.id == message.id then
                        { header | seen = True }

                    else
                        header

                map f messageList =
                    { messageList | headers = List.map f messageList.headers }
            in
            ( { model
                | state =
                    ShowingList (map updateSeen list)
                        (ShowingMessage
                            { visible
                                | message = { message | seen = True }
                                , markSeenAt = Nothing
                            }
                        )
              }
            , Api.markMessageSeen MarkedSeen message.mailbox message.id
            , Session.None
            )

        _ ->
            ( model, Cmd.none, Session.none )


updateOpenMessage : Session -> Model -> String -> ( Model, Cmd Msg, Session.Msg )
updateOpenMessage session model id =
    ( updateSelected model id
    , Api.getMessage MessageLoaded model.mailboxName id
    , Session.AddRecent model.mailboxName
    )



-- VIEW


view : Session -> Model -> { title : String, modal : Maybe (Html Msg), content : List (Html Msg) }
view session model =
    { title = model.mailboxName ++ " - Inbucket"
    , modal = viewModal model.promptPurge
    , content =
        [ div [ class "mailbox" ]
            [ aside [ class "message-list-controls" ]
                [ input
                    [ type_ "search"
                    , placeholder "search"
                    , onInput OnSearchInput
                    , value model.searchInput
                    ]
                    []
                , button [ onClick PurgeMailboxPrompt ] [ text "Purge" ]
                ]
            , viewMessageList session model
            , main_
                [ class "message" ]
                [ case model.state of
                    ShowingList _ NoMessage ->
                        text
                            ("Select a message on the left,"
                                ++ " or enter a different username into the box on upper right."
                            )

                    ShowingList _ (ShowingMessage { message }) ->
                        viewMessage session.zone message model.bodyMode

                    ShowingList _ (Transitioning { message }) ->
                        viewMessage session.zone message model.bodyMode

                    _ ->
                        text ""
                ]
            ]
        ]
    }


viewModal : Bool -> Maybe (Html Msg)
viewModal promptPurge =
    if promptPurge then
        Just <|
            div []
                [ p [] [ text "Are you sure you want to delete all messages in this mailbox?" ]
                , div [ class "button-bar" ]
                    [ button [ onClick PurgeMailboxConfirmed, class "danger" ] [ text "Yes" ]
                    , button [ onClick PurgeMailboxCanceled ] [ text "Cancel" ]
                    ]
                ]

    else
        Nothing


viewMessageList : Session -> Model -> Html Msg
viewMessageList session model =
    aside [ class "message-list" ] <|
        case model.state of
            LoadingList _ ->
                []

            ShowingList list _ ->
                list
                    |> filterMessageList
                    |> List.reverse
                    |> List.map (messageChip model list.selected)


messageChip : Model -> Maybe MessageID -> MessageHeader -> Html Msg
messageChip model selected message =
    div
        [ classList
            [ ( "message-list-entry", True )
            , ( "selected", selected == Just message.id )
            , ( "unseen", not message.seen )
            ]
        , onClick (ClickMessage message.id)
        ]
        [ div [ class "subject" ] [ text message.subject ]
        , div [ class "from" ] [ text message.from ]
        , div [ class "date" ] [ relativeDate model message.date ]
        ]


viewMessage : Time.Zone -> Message -> Body -> Html Msg
viewMessage zone message bodyMode =
    let
        sourceUrl =
            Api.serveUrl [ "mailbox", message.mailbox, message.id, "source" ]
    in
    div []
        [ div [ class "button-bar" ]
            [ button [ class "danger", onClick (DeleteMessage message) ] [ text "Delete" ]
            , a
                [ href sourceUrl, target "_blank" ]
                [ button [] [ text "Source" ] ]
            ]
        , dl [ class "message-header" ]
            [ dt [] [ text "From:" ]
            , dd [] [ text message.from ]
            , dt [] [ text "To:" ]
            , dd [] (List.map text message.to)
            , dt [] [ text "Date:" ]
            , dd [] [ verboseDate zone message.date ]
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
                , href "#"
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
                    Html.node "rendered-html" [ property "content" (Encode.string message.html) ] []

                TextBody ->
                    Html.node "rendered-html" [ property "content" (Encode.string message.text) ] []
            ]
        ]


attachments : Message -> Html Msg
attachments message =
    if List.isEmpty message.attachments then
        div [] []

    else
        table [ class "attachments well" ] (List.map (attachmentRow message) message.attachments)


attachmentRow : Message -> Message.Attachment -> Html Msg
attachmentRow message attach =
    let
        url =
            Api.serveUrl
                [ "mailbox"
                , message.mailbox
                , message.id
                , "attach"
                , attach.id
                , attach.fileName
                ]
    in
    tr []
        [ td []
            [ a [ href url, target "_blank" ] [ text attach.fileName ]
            , text (" (" ++ attach.contentType ++ ") ")
            ]
        , td [] [ a [ href url, download attach.fileName, class "button" ] [ text "Download" ] ]
        ]


relativeDate : Model -> Posix -> Html Msg
relativeDate model date =
    Relative.relativeTime model.now date |> text


verboseDate : Time.Zone -> Posix -> Html Msg
verboseDate zone date =
    text <|
        DF.format
            [ DF.monthNameFull
            , DF.text " "
            , DF.dayOfMonthSuffix
            , DF.text ", "
            , DF.yearNumber
            , DF.text " "
            , DF.hourNumber
            , DF.text ":"
            , DF.minuteFixed
            , DF.text ":"
            , DF.secondFixed
            , DF.text " "
            , DF.amPmUppercase
            , DF.text " (Local)"
            ]
            zone
            date



-- UTILITY


filterMessageList : MessageList -> List MessageHeader
filterMessageList list =
    if list.searchFilter == "" then
        list.headers

    else
        let
            matches header =
                String.contains list.searchFilter (String.toLower header.subject)
                    || String.contains list.searchFilter (String.toLower header.from)
        in
        List.filter matches list.headers
