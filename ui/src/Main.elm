module Main exposing
    ( Model
    , Msg(..)
    , Page(..)
    , applySession
    , init
    , main
    , pageSubscriptions
    , sessionChange
    , setRoute
    , subscriptions
    , update
    , updatePage
    , view
    )

import Browser exposing (Document, UrlRequest)
import Browser.Navigation as Nav
import Data.Session as Session exposing (Session, decoder)
import Html exposing (..)
import Json.Decode as D exposing (Value)
import Page.Home as Home
import Page.Mailbox as Mailbox
import Page.Monitor as Monitor
import Page.Status as Status
import Ports
import Route exposing (Route)
import Url exposing (Url)
import Views.Page as Page exposing (ActivePage(..), frame)



-- MODEL


type Page
    = Home Home.Model
    | Mailbox Mailbox.Model
    | Monitor Monitor.Model
    | Status Status.Model


type alias Model =
    { page : Page
    , session : Session
    , mailboxName : String
    }


init : Value -> Url -> Nav.Key -> ( Model, Cmd Msg )
init sessionValue location key =
    let
        session =
            Session.init key location (Session.decodeValueWithDefault sessionValue)

        ( subModel, _ ) =
            Home.init

        model =
            { page = Home subModel
            , session = session
            , mailboxName = ""
            }

        route =
            Route.fromUrl location
    in
    applySession (setRoute route model)


type Msg
    = SetRoute Route
    | UrlChanged Url
    | LinkClicked UrlRequest
    | UpdateSession (Result D.Error Session.Persistent)
    | MailboxNameInput String
    | ViewMailbox String
    | HomeMsg Home.Msg
    | MailboxMsg Mailbox.Msg
    | MonitorMsg Monitor.Msg
    | StatusMsg Status.Msg



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ pageSubscriptions model.page
        , Sub.map UpdateSession sessionChange
        ]


sessionChange : Sub (Result D.Error Session.Persistent)
sessionChange =
    Ports.onSessionChange (D.decodeValue Session.decoder)


pageSubscriptions : Page -> Sub Msg
pageSubscriptions page =
    case page of
        Mailbox subModel ->
            Sub.map MailboxMsg (Mailbox.subscriptions subModel)

        Monitor subModel ->
            Sub.map MonitorMsg (Monitor.subscriptions subModel)

        Status subModel ->
            Sub.map StatusMsg (Status.subscriptions subModel)

        _ ->
            Sub.none



-- UPDATE


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    applySession <|
        case msg of
            LinkClicked req ->
                case req of
                    Browser.Internal url ->
                        ( model, Nav.pushUrl model.session.key (Url.toString url), Session.none )

                    _ ->
                        Debug.todo "implement external links"

            UrlChanged url ->
                -- Responds to new browser URL.
                if model.session.routing then
                    setRoute (Route.fromUrl url) model

                else
                    -- Skip once, but re-enable routing.
                    ( model, Cmd.none, Session.EnableRouting )

            SetRoute route ->
                -- Updates broser URL to requested route.
                ( model, Route.newUrl model.session.key route, Session.none )

            UpdateSession (Ok persistent) ->
                let
                    session =
                        model.session
                in
                ( { model | session = { session | persistent = persistent } }
                , Cmd.none
                , Session.none
                )

            UpdateSession (Err error) ->
                let
                    _ =
                        Debug.log "Error decoding session" error
                in
                ( model, Cmd.none, Session.none )

            MailboxNameInput name ->
                ( { model | mailboxName = name }, Cmd.none, Session.none )

            ViewMailbox name ->
                ( { model | mailboxName = "" }
                , Route.newUrl model.session.key (Route.Mailbox name)
                , Session.none
                )

            _ ->
                updatePage msg model


{-| Delegates incoming messages to their respective sub-pages.
-}
updatePage : Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
updatePage msg model =
    let
        -- Handles sub-model update by calling toUpdate with subMsg & subModel, then packing the
        -- updated sub-model back into model.page.
        modelUpdate toPage toMsg subUpdate subMsg subModel =
            let
                ( newModel, subCmd, sessionMsg ) =
                    subUpdate model.session subMsg subModel
            in
            ( { model | page = toPage newModel }, Cmd.map toMsg subCmd, sessionMsg )
    in
    case ( msg, model.page ) of
        ( HomeMsg subMsg, Home subModel ) ->
            modelUpdate Home HomeMsg Home.update subMsg subModel

        ( MailboxMsg subMsg, Mailbox subModel ) ->
            modelUpdate Mailbox MailboxMsg Mailbox.update subMsg subModel

        ( MonitorMsg subMsg, Monitor subModel ) ->
            modelUpdate Monitor MonitorMsg Monitor.update subMsg subModel

        ( StatusMsg subMsg, Status subModel ) ->
            modelUpdate Status StatusMsg Status.update subMsg subModel

        ( _, _ ) ->
            -- Disregard messages destined for the wrong page.
            ( model, Cmd.none, Session.none )


setRoute : Route -> Model -> ( Model, Cmd Msg, Session.Msg )
setRoute route model =
    let
        ( newModel, newCmd, newSession ) =
            case route of
                Route.Unknown hash ->
                    ( model, Cmd.none, Session.SetFlash ("Unknown route requested: " ++ hash) )

                Route.Home ->
                    let
                        ( subModel, subCmd ) =
                            Home.init
                    in
                    ( { model | page = Home subModel }
                    , Cmd.map HomeMsg subCmd
                    , Session.none
                    )

                Route.Mailbox name ->
                    let
                        ( subModel, subCmd ) =
                            Mailbox.init name Nothing
                    in
                    ( { model | page = Mailbox subModel }
                    , Cmd.map MailboxMsg subCmd
                    , Session.none
                    )

                Route.Message mailbox id ->
                    let
                        ( subModel, subCmd ) =
                            Mailbox.init mailbox (Just id)
                    in
                    ( { model | page = Mailbox subModel }
                    , Cmd.map MailboxMsg subCmd
                    , Session.none
                    )

                Route.Monitor ->
                    let
                        ( subModel, subCmd ) =
                            Monitor.init
                    in
                    ( { model | page = Monitor subModel }
                    , Cmd.map MonitorMsg subCmd
                    , Session.none
                    )

                Route.Status ->
                    ( { model | page = Status Status.init }
                    , Cmd.batch
                        [ Ports.windowTitle "Inbucket Status"
                        , Cmd.map StatusMsg Status.load
                        ]
                    , Session.none
                    )
    in
    case model.page of
        Monitor _ ->
            -- Leaving Monitor page, shut down the web socket.
            ( newModel, Cmd.batch [ Ports.monitorCommand False, newCmd ], newSession )

        _ ->
            ( newModel, newCmd, newSession )


applySession : ( Model, Cmd Msg, Session.Msg ) -> ( Model, Cmd Msg )
applySession ( model, cmd, sessionMsg ) =
    let
        session =
            Session.update sessionMsg model.session

        newModel =
            { model | session = session }
    in
    if session.persistent == model.session.persistent then
        -- No change
        ( newModel, cmd )

    else
        ( newModel
        , Cmd.batch [ cmd, Ports.storeSession session.persistent ]
        )



-- VIEW


view : Model -> Document Msg
view model =
    let
        mailbox =
            case model.page of
                Mailbox subModel ->
                    subModel.mailboxName

                _ ->
                    ""

        controls =
            { viewMailbox = ViewMailbox
            , mailboxOnInput = MailboxNameInput
            , mailboxValue = model.mailboxName
            , recentOptions = model.session.persistent.recentMailboxes
            , recentActive = mailbox
            }

        frame =
            Page.frame controls model.session
    in
    Document "Inbucket Document"
        [ case model.page of
            Home subModel ->
                Html.map HomeMsg (Home.view model.session subModel)
                    |> frame Page.Other

            Mailbox subModel ->
                Html.map MailboxMsg (Mailbox.view model.session subModel)
                    |> frame Page.Mailbox

            Monitor subModel ->
                Html.map MonitorMsg (Monitor.view model.session subModel)
                    |> frame Page.Monitor

            Status subModel ->
                Html.map StatusMsg (Status.view model.session subModel)
                    |> frame Page.Status
        ]



-- MAIN


main : Program Value Model Msg
main =
    Browser.application
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        , onUrlChange = UrlChanged
        , onUrlRequest = LinkClicked
        }
