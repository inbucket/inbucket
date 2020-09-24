module Page.Mailbox exposing (Model, Msg, init, subscriptions, update, view)

import Api
import Data.Message as Message exposing (Message)
import Data.MessageHeader exposing (MessageHeader)
import Data.Session exposing (Session)
import DateFormat as DF
import DateFormat.Relative as Relative
import Effect exposing (Effect)
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
    , socketConnected : Bool
    , bodyMode : Body
    , searchInput : String
    , promptPurge : Bool
    , markSeenTimer : Timer
    , now : Posix
    }


type alias ServeUrl =
    List String -> String


init : Session -> String -> Maybe MessageID -> ( Model, Effect Msg )
init session mailboxName selection =
    ( { session = session
      , mailboxName = mailboxName
      , state = LoadingList selection
      , socketConnected = False
      , bodyMode = SafeHtmlBody
      , searchInput = ""
      , promptPurge = False
      , markSeenTimer = Timer.empty
      , now = Time.millisToPosix 0
      }
    , Effect.batch
        [ Effect.posixTime Tick
        , Effect.getHeaderList ListLoaded mailboxName
        ]
    )



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions _ =
    Time.every (30 * 1000) Tick



-- UPDATE


type Msg
    = ListLoaded (Result HttpUtil.Error (List MessageHeader))
    | ClickMessage MessageID
    | ClickRefresh
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


update : Msg -> Model -> ( Model, Effect Msg )
update msg model =
    case msg of
        ClickMessage id ->
            ( updateSelected model id
            , Effect.batch
                [ -- Update browser location.
                  Effect.updateRoute (Route.Message model.mailboxName id)
                , Effect.getMessage MessageLoaded model.mailboxName id
                ]
            )

        ClickRefresh ->
            let
                selection =
                    case model.state of
                        ShowingList _ (ShowingMessage message) ->
                            Just message.id

                        _ ->
                            Nothing
            in
            -- Reset to loading state, preserving the current message selection.
            ( { model | state = LoadingList selection }
            , Effect.getHeaderList ListLoaded model.mailboxName
            )

        CloseMessage ->
            case model.state of
                ShowingList list _ ->
                    ( { model | state = ShowingList list NoMessage }, Effect.none )

                _ ->
                    ( model, Effect.none )

        DeleteMessage message ->
            updateDeleteMessage model message

        DeletedMessage (Ok _) ->
            ( model, Effect.none )

        DeletedMessage (Err err) ->
            ( model, Effect.showFlash (HttpUtil.errorFlash err) )

        ListKeyPress id keyCode ->
            case keyCode of
                13 ->
                    updateOpenMessage model id

                _ ->
                    ( model, Effect.none )

        ListLoaded (Ok headers) ->
            updateListLoaded model headers

        ListLoaded (Err err) ->
            ( model, Effect.showFlash (HttpUtil.errorFlash err) )

        MarkSeenLoaded (Ok _) ->
            ( model, Effect.none )

        MarkSeenLoaded (Err err) ->
            ( model, Effect.showFlash (HttpUtil.errorFlash err) )

        MessageLoaded (Ok message) ->
            updateMessageResult model message

        MessageLoaded (Err err) ->
            ( model, Effect.showFlash (HttpUtil.errorFlash err) )

        MessageBody bodyMode ->
            ( { model | bodyMode = bodyMode }, Effect.none )

        ModalFocused message ->
            ( model, Effect.focusModalResult message )

        OnSearchInput searchInput ->
            updateSearchInput model searchInput

        PurgeMailboxPrompt ->
            ( { model | promptPurge = True }, Effect.focusModal ModalFocused )

        PurgeMailboxCanceled ->
            ( { model | promptPurge = False }, Effect.none )

        PurgeMailboxConfirmed ->
            updateTriggerPurge model

        PurgedMailbox (Ok _) ->
            ( model, Effect.none )

        PurgedMailbox (Err err) ->
            ( model, Effect.showFlash (HttpUtil.errorFlash err) )

        MarkSeenTriggered timer ->
            if timer == model.markSeenTimer then
                -- Matching timer means we have changed messages, mark this one seen.
                updateMarkMessageSeen model

            else
                ( model, Effect.none )

        Tick now ->
            ( { model | now = now }, Effect.none )


updateListLoaded : Model -> List MessageHeader -> ( Model, Effect Msg )
updateListLoaded model headers =
    case model.state of
        LoadingList selection ->
            let
                newModel =
                    { model
                        | state = ShowingList (MessageList headers Nothing "") NoMessage
                    }
            in
            Effect.append (Effect.addRecent newModel.mailboxName) <|
                case selection of
                    Just id ->
                        -- Don't try to load selected message if not present in headers.
                        if List.any (\header -> Just header.id == selection) headers then
                            updateOpenMessage newModel id

                        else
                            ( newModel, Effect.updateRoute (Route.Mailbox model.mailboxName) )

                    Nothing ->
                        ( newModel, Effect.none )

        _ ->
            ( model, Effect.none )


{-| Replace the currently displayed message.
-}
updateMessageResult : Model -> Message -> ( Model, Effect Msg )
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
            ( model, Effect.none )

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
            , Effect.schedule MarkSeenTriggered newTimer 1500
            )


{-| Updates model and triggers commands to purge this mailbox.
-}
updateTriggerPurge : Model -> ( Model, Effect Msg )
updateTriggerPurge model =
    ( { model
        | promptPurge = False
        , state = ShowingList (MessageList [] Nothing "") NoMessage
      }
    , Effect.batch
        [ Effect.updateRoute (Route.Mailbox model.mailboxName)
        , Effect.purgeMailbox PurgedMailbox model.mailboxName
        ]
    )


updateSearchInput : Model -> String -> ( Model, Effect Msg )
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
            ( model, Effect.none )

        ShowingList list messageState ->
            ( { model
                | searchInput = searchInput
                , state = ShowingList { list | searchFilter = searchFilter } messageState
              }
            , Effect.none
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


updateDeleteMessage : Model -> Message -> ( Model, Effect Msg )
updateDeleteMessage model message =
    let
        filter f messageList =
            { messageList | headers = List.filter f messageList.headers }
    in
    case model.state of
        ShowingList list _ ->
            ( { model | state = ShowingList (filter (\x -> x.id /= message.id) list) NoMessage }
            , Effect.batch
                [ Effect.deleteMessage DeletedMessage message.mailbox message.id
                , Effect.updateRoute (Route.Mailbox model.mailboxName)
                ]
            )

        _ ->
            ( model, Effect.none )


{-| Updates both the active message, and the message list to mark the currently viewed message as seen.
-}
updateMarkMessageSeen : Model -> ( Model, Effect Msg )
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
            , Effect.markMessageSeen MarkSeenLoaded visibleMessage.mailbox visibleMessage.id
            )

        _ ->
            ( model, Effect.none )


updateOpenMessage : Model -> String -> ( Model, Effect Msg )
updateOpenMessage model id =
    ( updateSelected model id
    , Effect.getMessage MessageLoaded model.mailboxName id
    )



-- VIEW


view : Model -> { title : String, modal : Maybe (Html Msg), content : List (Html Msg) }
view model =
    let
        serveUrl : ServeUrl
        serveUrl =
            Api.serveUrl model.session

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
            [ viewMessageListControls model
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
                        viewMessage serveUrl model.session.zone message model.bodyMode

                    ShowingList _ (Transitioning message) ->
                        viewMessage serveUrl model.session.zone message model.bodyMode

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


viewMessageListControls : Model -> Html Msg
viewMessageListControls model =
    let
        clearButton =
            Just <|
                button
                    [ Events.onClick (OnSearchInput "")
                    , disabled (model.searchInput == "")
                    , alt "Clear Search"
                    ]
                    [ i [ class "fas fa-times" ] [] ]

        purgeButton =
            Just <|
                button
                    [ Events.onClick PurgeMailboxPrompt
                    , alt "Purge Mailbox"
                    ]
                    [ i [ class "fas fa-trash" ] [] ]

        refreshButton =
            if model.socketConnected then
                Nothing

            else
                Just <|
                    button
                        [ Events.onClick ClickRefresh
                        , alt "Refresh Mailbox"
                        ]
                        [ i [ class "fas fa-sync" ] [] ]

        searchInput =
            Just <|
                input
                    [ type_ "text"
                    , placeholder "search"
                    , Events.onInput OnSearchInput
                    , value model.searchInput
                    ]
                    []
    in
    [ searchInput, clearButton, refreshButton, purgeButton ]
        |> List.filterMap identity
        |> aside [ class "message-list-controls" ]


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


viewMessage : ServeUrl -> Time.Zone -> Message -> Body -> Html Msg
viewMessage serveUrl zone message bodyMode =
    let
        htmlUrl =
            serveUrl [ "mailbox", message.mailbox, message.id, "html" ]

        sourceUrl =
            serveUrl [ "mailbox", message.mailbox, message.id, "source" ]

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
        , attachments serveUrl message
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


attachments : ServeUrl -> Message -> Html Msg
attachments serveUrl message =
    if List.isEmpty message.attachments then
        text ""

    else
        message.attachments
            |> List.map (attachmentRow serveUrl message)
            |> table [ class "attachments well" ]


attachmentRow : ServeUrl -> Message -> Message.Attachment -> Html Msg
attachmentRow serveUrl message attach =
    let
        url =
            serveUrl
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
