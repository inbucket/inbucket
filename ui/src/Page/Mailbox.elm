module Page.Mailbox exposing (Model, Msg, init, load, subscriptions, update, view)

import Api
import Browser.Navigation as Nav
import Data.Message as Message exposing (Message)
import Data.MessageHeader exposing (MessageHeader)
import Data.Session as Session exposing (Session)
import DateFormat as DF
import DateFormat.Relative as Relative
import Html
    exposing
        ( Attribute
        , Html
        , a
        , article
        , aside
        , button
        , dd
        , div
        , dl
        , dt
        , h3
        , i
        , input
        , li
        , main_
        , nav
        , p
        , span
        , table
        , td
        , text
        , tr
        , ul
        )
import Html.Attributes
    exposing
        ( alt
        , class
        , classList
        , disabled
        , download
        , href
        , placeholder
        , property
        , tabindex
        , target
        , type_
        , value
        )
import Html.Events as Events
import HttpUtil
import Json.Decode as D
import Json.Encode as E
import Modal
import Route
import Task
import Time exposing (Posix)
import Timer exposing (Timer)



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
    | ShowingMessage Message
    | Transitioning Message


type alias MessageID =
    String


type alias MessageList =
    { headers : List MessageHeader
    , selected : Maybe MessageID
    , searchFilter : String
    }


type alias Model =
    { session : Session
    , mailboxName : String
    , state : State
    , bodyMode : Body
    , searchInput : String
    , promptPurge : Bool
    , markSeenTimer : Timer
    , now : Posix
    }


init : Session -> String -> Maybe MessageID -> ( Model, Cmd Msg )
init session mailboxName selection =
    ( { session = session
      , mailboxName = mailboxName
      , state = LoadingList selection
      , bodyMode = SafeHtmlBody
      , searchInput = ""
      , promptPurge = False
      , markSeenTimer = Timer.empty
      , now = Time.millisToPosix 0
      }
    , load mailboxName
    )


load : String -> Cmd Msg
load mailboxName =
    Cmd.batch
        [ Task.perform Tick Time.now
        , Api.getHeaderList ListLoaded mailboxName
        ]



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions _ =
    Time.every (30 * 1000) Tick



-- UPDATE


type Msg
    = ListLoaded (Result HttpUtil.Error (List MessageHeader))
    | ClickMessage MessageID
    | ListKeyPress String Int
    | CloseMessage
    | MessageLoaded (Result HttpUtil.Error Message)
    | MessageBody Body
    | MarkSeenTriggered Timer
    | MarkSeenLoaded (Result HttpUtil.Error ())
    | DeleteMessage Message
    | DeletedMessage (Result HttpUtil.Error ())
    | PurgeMailboxPrompt
    | PurgeMailboxCanceled
    | PurgeMailboxConfirmed
    | PurgedMailbox (Result HttpUtil.Error ())
    | OnSearchInput String
    | Tick Posix
    | ModalFocused Modal.Msg


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        ClickMessage id ->
            ( updateSelected { model | session = Session.disableRouting model.session } id
            , Cmd.batch
                [ -- Update browser location.
                  Route.Message model.mailboxName id
                    |> model.session.router.toPath
                    |> Nav.replaceUrl model.session.key
                , Api.getMessage MessageLoaded model.mailboxName id
                ]
            )

        CloseMessage ->
            case model.state of
                ShowingList list _ ->
                    ( { model | state = ShowingList list NoMessage }, Cmd.none )

                _ ->
                    ( model, Cmd.none )

        DeleteMessage message ->
            updateDeleteMessage model message

        DeletedMessage (Ok _) ->
            ( model, Cmd.none )

        DeletedMessage (Err err) ->
            ( { model | session = Session.showFlash (HttpUtil.errorFlash err) model.session }
            , Cmd.none
            )

        ListKeyPress id keyCode ->
            case keyCode of
                13 ->
                    updateOpenMessage model id

                _ ->
                    ( model, Cmd.none )

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
                            updateOpenMessage newModel id

                        Nothing ->
                            ( { newModel
                                | session = Session.addRecent model.mailboxName model.session
                              }
                            , Cmd.none
                            )

                _ ->
                    ( model, Cmd.none )

        ListLoaded (Err err) ->
            ( { model | session = Session.showFlash (HttpUtil.errorFlash err) model.session }
            , Cmd.none
            )

        MarkSeenLoaded (Ok _) ->
            ( model, Cmd.none )

        MarkSeenLoaded (Err err) ->
            ( { model | session = Session.showFlash (HttpUtil.errorFlash err) model.session }
            , Cmd.none
            )

        MessageLoaded (Ok message) ->
            updateMessageResult model message

        MessageLoaded (Err err) ->
            ( { model | session = Session.showFlash (HttpUtil.errorFlash err) model.session }
            , Cmd.none
            )

        MessageBody bodyMode ->
            ( { model | bodyMode = bodyMode }, Cmd.none )

        ModalFocused message ->
            ( { model | session = Modal.updateSession message model.session }
            , Cmd.none
            )

        OnSearchInput searchInput ->
            updateSearchInput model searchInput

        PurgeMailboxPrompt ->
            ( { model | promptPurge = True }, Modal.resetFocusCmd ModalFocused )

        PurgeMailboxCanceled ->
            ( { model | promptPurge = False }, Cmd.none )

        PurgeMailboxConfirmed ->
            updateTriggerPurge model

        PurgedMailbox (Ok _) ->
            ( model, Cmd.none )

        PurgedMailbox (Err err) ->
            ( { model | session = Session.showFlash (HttpUtil.errorFlash err) model.session }
            , Cmd.none
            )

        MarkSeenTriggered timer ->
            if timer == model.markSeenTimer then
                -- Matching timer means we have changed messages, mark this one seen.
                updateMarkMessageSeen model

            else
                ( model, Cmd.none )

        Tick now ->
            ( { model | now = now }, Cmd.none )


{-| Replace the currently displayed message.
-}
updateMessageResult : Model -> Message -> ( Model, Cmd Msg )
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
            ( model, Cmd.none )

        ShowingList list _ ->
            let
                newTimer =
                    Timer.replace model.markSeenTimer
            in
            ( { model
                | state =
                    ShowingList
                        { list | selected = Just message.id }
                        (ShowingMessage message)
                , bodyMode = bodyMode
                , markSeenTimer = newTimer
              }
              -- Set 1500ms delay before reporting message as seen to backend.
            , Timer.schedule MarkSeenTriggered newTimer 1500
            )


{-| Updates model and triggers commands to purge this mailbox.
-}
updateTriggerPurge : Model -> ( Model, Cmd Msg )
updateTriggerPurge model =
    let
        cmd =
            Cmd.batch
                [ Route.Mailbox model.mailboxName
                    |> model.session.router.toPath
                    |> Nav.replaceUrl model.session.key
                , Api.purgeMailbox PurgedMailbox model.mailboxName
                ]
    in
    case model.state of
        ShowingList _ _ ->
            ( { model
                | promptPurge = False
                , session = Session.disableRouting model.session
                , state = ShowingList (MessageList [] Nothing "") NoMessage
              }
            , cmd
            )

        _ ->
            ( model, cmd )


updateSearchInput : Model -> String -> ( Model, Cmd Msg )
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
            ( model, Cmd.none )

        ShowingList list messageState ->
            ( { model
                | searchInput = searchInput
                , state = ShowingList { list | searchFilter = searchFilter } messageState
              }
            , Cmd.none
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


updateDeleteMessage : Model -> Message -> ( Model, Cmd Msg )
updateDeleteMessage model message =
    let
        filter f messageList =
            { messageList | headers = List.filter f messageList.headers }
    in
    case model.state of
        ShowingList list _ ->
            ( { model
                | session = Session.disableRouting model.session
                , state =
                    ShowingList (filter (\x -> x.id /= message.id) list) NoMessage
              }
            , Cmd.batch
                [ Api.deleteMessage DeletedMessage message.mailbox message.id
                , Route.Mailbox model.mailboxName
                    |> model.session.router.toPath
                    |> Nav.replaceUrl model.session.key
                ]
            )

        _ ->
            ( model, Cmd.none )


{-| Updates both the active message, and the message list to mark the currently viewed message as seen.
-}
updateMarkMessageSeen : Model -> ( Model, Cmd Msg )
updateMarkMessageSeen model =
    case model.state of
        ShowingList messages (ShowingMessage visibleMessage) ->
            let
                updateHeader header =
                    if header.id == visibleMessage.id then
                        { header | seen = True }

                    else
                        header

                newMessages =
                    { messages | headers = List.map updateHeader messages.headers }
            in
            ( { model
                | state =
                    ShowingList newMessages (ShowingMessage { visibleMessage | seen = True })
              }
            , Api.markMessageSeen MarkSeenLoaded visibleMessage.mailbox visibleMessage.id
            )

        _ ->
            ( model, Cmd.none )


updateOpenMessage : Model -> String -> ( Model, Cmd Msg )
updateOpenMessage model id =
    let
        newModel =
            { model | session = Session.addRecent model.mailboxName model.session }
    in
    ( updateSelected newModel id
    , Api.getMessage MessageLoaded model.mailboxName id
    )



-- VIEW


view : Model -> { title : String, modal : Maybe (Html Msg), content : List (Html Msg) }
view model =
    let
        mode =
            case model.state of
                ShowingList _ (ShowingMessage _) ->
                    "message-active"

                _ ->
                    "list-active"
    in
    { title = model.mailboxName ++ " - Inbucket"
    , modal = viewModal model.promptPurge
    , content =
        [ div [ class ("mailbox " ++ mode) ]
            [ aside [ class "message-list-controls" ]
                [ input
                    [ type_ "text"
                    , placeholder "search"
                    , Events.onInput OnSearchInput
                    , value model.searchInput
                    ]
                    []
                , button
                    [ Events.onClick (OnSearchInput "")
                    , disabled (model.searchInput == "")
                    , alt "Clear Search"
                    ]
                    [ i [ class "fas fa-times" ] [] ]
                , button
                    [ Events.onClick PurgeMailboxPrompt
                    , alt "Purge Mailbox"
                    ]
                    [ i [ class "fas fa-trash" ] [] ]
                ]
            , viewMessageList model
            , main_
                [ class "message" ]
                [ case model.state of
                    ShowingList _ NoMessage ->
                        text
                            ("Select a message on the left,"
                                ++ " or enter a different username into the box on upper right."
                            )

                    ShowingList _ (ShowingMessage message) ->
                        viewMessage model.session.zone message model.bodyMode

                    ShowingList _ (Transitioning message) ->
                        viewMessage model.session.zone message model.bodyMode

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
                    [ button [ Events.onClick PurgeMailboxConfirmed, class "danger" ] [ text "Yes" ]
                    , button [ Events.onClick PurgeMailboxCanceled ] [ text "Cancel" ]
                    ]
                ]

    else
        Nothing


viewMessageList : Model -> Html Msg
viewMessageList model =
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
        [ class "message-list-entry"
        , classList
            [ ( "selected", selected == Just message.id )
            , ( "unseen", not message.seen )
            ]
        , Events.onClick (ClickMessage message.id)
        , onKeyUp (ListKeyPress message.id)
        , tabindex 0
        ]
        [ div [ class "subject" ] [ text message.subject ]
        , div [ class "from" ] [ text message.from ]
        , div [ class "date" ] [ relativeDate model message.date ]
        ]


viewMessage : Time.Zone -> Message -> Body -> Html Msg
viewMessage zone message bodyMode =
    let
        htmlUrl =
            Api.serveUrl [ "mailbox", message.mailbox, message.id, "html" ]

        sourceUrl =
            Api.serveUrl [ "mailbox", message.mailbox, message.id, "source" ]

        htmlButton =
            if message.html == "" then
                text ""

            else
                a [ href htmlUrl, target "_blank" ]
                    [ button [ tabindex -1 ] [ text "Raw HTML" ] ]
    in
    div []
        [ div [ class "button-bar" ]
            [ button [ class "message-close light", Events.onClick CloseMessage ]
                [ i [ class "fas fa-arrow-left" ] [] ]
            , button [ class "danger", Events.onClick (DeleteMessage message) ] [ text "Delete" ]
            , a [ href sourceUrl, target "_blank" ]
                [ button [ tabindex -1 ] [ text "Source" ] ]
            , htmlButton
            ]
        , dl [ class "message-header" ]
            [ dt [] [ text "From:" ]
            , dd [] [ text message.from ]
            , dt [] [ text "To:" ]
            , dd [] [ text (String.join ", " message.to) ]
            , dt [] [ text "Date:" ]
            , dd [] [ verboseDate zone message.date ]
            , dt [] [ text "Subject:" ]
            , dd [] [ text message.subject ]
            ]
        , messageErrors message
        , messageBody message bodyMode
        , attachments message
        ]


messageErrors : Message -> Html Msg
messageErrors message =
    let
        row error =
            li []
                [ span
                    [ classList [ ( "message-warn-severe", error.severe ) ] ]
                    [ text (error.name ++ ": ") ]
                , text error.detail
                ]
    in
    case message.errors of
        [] ->
            text ""

        errors ->
            div [ class "well well-warn message-warn" ]
                [ div [] [ h3 [] [ text "MIME problems detected" ] ]
                , ul [] (List.map row errors)
                ]


messageBody : Message -> Body -> Html Msg
messageBody message bodyMode =
    let
        bodyModeTab mode label =
            a
                [ classList [ ( "active", bodyMode == mode ) ]
                , Events.onClick (MessageBody mode)
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
                    Html.node "rendered-html" [ property "content" (E.string message.html) ] []

                TextBody ->
                    Html.node "rendered-html" [ property "content" (E.string message.text) ] []
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


onKeyUp : (Int -> msg) -> Attribute msg
onKeyUp tagger =
    Events.on "keyup" (D.map tagger Events.keyCode)
