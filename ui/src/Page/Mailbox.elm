module Page.Mailbox exposing (Model, Msg, init, load, subscriptions, update, view)

import Data.Message as Message exposing (Message)
import Data.MessageHeader as MessageHeader exposing (MessageHeader)
import Data.Session as Session exposing (Session)
import Date exposing (Date)
import DateFormat
import DateFormat.Relative as Relative
import Html exposing (..)
import Html.Attributes
    exposing
        ( class
        , classList
        , downloadAs
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
import Time exposing (Time)


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
    , markSeenAt : Maybe Time
    }


type alias Model =
    { mailboxName : String
    , state : State
    , bodyMode : Body
    , searchInput : String
    , now : Date
    }


init : String -> Maybe MessageID -> ( Model, Cmd Msg )
init mailboxName selection =
    ( Model mailboxName (LoadingList selection) SafeHtmlBody "" (Date.fromTime 0)
    , load mailboxName
    )


load : String -> Cmd Msg
load mailboxName =
    Cmd.batch
        [ Ports.windowTitle (mailboxName ++ " - Inbucket")
        , Task.perform Tick Time.now
        , getList mailboxName
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
                        Time.every (250 * Time.millisecond) SeenTick

                _ ->
                    Sub.none
    in
    Sub.batch
        [ Time.every (30 * Time.second) Tick
        , subSeen
        ]



-- UPDATE


type Msg
    = ClickMessage MessageID
    | DeleteMessage Message
    | DeleteMessageResult (Result Http.Error ())
    | ListResult (Result Http.Error (List MessageHeader))
    | MarkSeenResult (Result Http.Error ())
    | MessageResult (Result Http.Error Message)
    | MessageBody Body
    | OpenedTime Time
    | Purge
    | PurgeResult (Result Http.Error ())
    | SearchInput String
    | SeenTick Time
    | Tick Time
    | ViewMessage MessageID


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    case msg of
        ClickMessage id ->
            ( updateSelected model id
            , Cmd.batch
                [ -- Update browser location.
                  Route.newUrl (Route.Message model.mailboxName id)
                , getMessage model.mailboxName id
                ]
            , Session.DisableRouting
            )

        ViewMessage id ->
            ( updateSelected model id
            , getMessage model.mailboxName id
            , Session.AddRecent model.mailboxName
            )

        DeleteMessage message ->
            updateDeleteMessage model message

        DeleteMessageResult (Ok _) ->
            ( model, Cmd.none, Session.none )

        DeleteMessageResult (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )

        ListResult (Ok headers) ->
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

        MessageResult (Ok message) ->
            updateMessageResult model message

        MessageResult (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )

        MessageBody bodyMode ->
            ( { model | bodyMode = bodyMode }, Cmd.none, Session.none )

        SearchInput searchInput ->
            updateSearchInput model searchInput

        OpenedTime time ->
            case model.state of
                ShowingList list (ShowingMessage visible) ->
                    if visible.message.seen then
                        ( model, Cmd.none, Session.none )
                    else
                        -- Set delay before reporting message as seen to backend.
                        ( { model
                            | state =
                                ShowingList list
                                    (ShowingMessage
                                        { visible
                                            | markSeenAt = Just (time + (1.5 * Time.second))
                                        }
                                    )
                          }
                        , Cmd.none
                        , Session.none
                        )

                _ ->
                    ( model, Cmd.none, Session.none )

        Purge ->
            updatePurge model

        PurgeResult (Ok _) ->
            ( model, Cmd.none, Session.none )

        PurgeResult (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )

        SeenTick now ->
            case model.state of
                ShowingList _ (ShowingMessage { message, markSeenAt }) ->
                    case markSeenAt of
                        Just deadline ->
                            if now >= deadline then
                                updateMarkMessageSeen model message
                            else
                                ( model, Cmd.none, Session.none )

                        Nothing ->
                            ( model, Cmd.none, Session.none )

                _ ->
                    ( model, Cmd.none, Session.none )

        Tick now ->
            ( { model | now = Date.fromTime now }, Cmd.none, Session.none )


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


updatePurge : Model -> ( Model, Cmd Msg, Session.Msg )
updatePurge model =
    let
        cmd =
            "/api/v1/mailbox/"
                ++ model.mailboxName
                |> HttpUtil.delete
                |> Http.send PurgeResult
    in
    case model.state of
        ShowingList list _ ->
            ( { model | state = ShowingList (MessageList [] Nothing "") NoMessage }
            , cmd
            , Session.none
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


updateDeleteMessage : Model -> Message -> ( Model, Cmd Msg, Session.Msg )
updateDeleteMessage model message =
    let
        url =
            "/api/v1/mailbox/" ++ message.mailbox ++ "/" ++ message.id

        cmd =
            HttpUtil.delete url
                |> Http.send DeleteMessageResult

        filter f messageList =
            { messageList | headers = List.filter f messageList.headers }
    in
    case model.state of
        ShowingList list _ ->
            ( { model
                | state =
                    ShowingList (filter (\x -> x.id /= message.id) list) NoMessage
              }
            , cmd
            , Session.none
            )

        _ ->
            ( model, cmd, Session.none )


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

                url =
                    "/api/v1/mailbox/" ++ message.mailbox ++ "/" ++ message.id

                command =
                    -- The URL tells the API what message to update, so we only need to indicate the
                    -- desired change in the body.
                    Encode.object [ ( "seen", Encode.bool True ) ]
                        |> Http.jsonBody
                        |> HttpUtil.patch url
                        |> Http.send MarkSeenResult

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
            , command
            , Session.None
            )

        _ ->
            ( model, Cmd.none, Session.none )


getList : String -> Cmd Msg
getList mailboxName =
    let
        url =
            "/api/v1/mailbox/" ++ mailboxName
    in
    Http.get url (Decode.list MessageHeader.decoder)
        |> Http.send ListResult


getMessage : String -> MessageID -> Cmd Msg
getMessage mailboxName id =
    let
        url =
            "/serve/m/" ++ mailboxName ++ "/" ++ id
    in
    Http.get url Message.decoder
        |> Http.send MessageResult



-- VIEW


view : Session -> Model -> Html Msg
view session model =
    div [ id "page", class "mailbox" ]
        [ viewMessageList session model
        , main_
            [ id "message" ]
            [ case model.state of
                ShowingList _ NoMessage ->
                    text
                        ("Select a message on the left,"
                            ++ " or enter a different username into the box on upper right."
                        )

                ShowingList _ (ShowingMessage { message }) ->
                    viewMessage message model.bodyMode

                ShowingList _ (Transitioning { message }) ->
                    viewMessage message model.bodyMode

                _ ->
                    text ""
            ]
        ]


viewMessageList : Session -> Model -> Html Msg
viewMessageList session model =
    aside [ id "message-list" ]
        [ div []
            [ input
                [ type_ "search"
                , placeholder "search"
                , onInput SearchInput
                , value model.searchInput
                ]
                []
            , button [ onClick Purge ] [ text "Purge" ]
            ]
        , case model.state of
            LoadingList _ ->
                div [] []

            ShowingList list _ ->
                div []
                    (list
                        |> filterMessageList
                        |> List.reverse
                        |> List.map (messageChip model list.selected)
                    )
        ]


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
            , dd [] [ verboseDate message.date ]
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


relativeDate : Model -> Date -> Html Msg
relativeDate model date =
    Relative.relativeTime model.now date |> text


verboseDate : Date -> Html Msg
verboseDate date =
    DateFormat.format
        [ DateFormat.monthNameFull
        , DateFormat.text " "
        , DateFormat.dayOfMonthSuffix
        , DateFormat.text ", "
        , DateFormat.yearNumber
        , DateFormat.text " "
        , DateFormat.hourNumber
        , DateFormat.text ":"
        , DateFormat.minuteFixed
        , DateFormat.text ":"
        , DateFormat.secondFixed
        , DateFormat.text " "
        , DateFormat.amPmUppercase
        ]
        date
        |> text



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
