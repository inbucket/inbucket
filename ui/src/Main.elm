module Main exposing (main)

import Browser exposing (Document, UrlRequest)
import Browser.Navigation as Nav
import Data.AppConfig as AppConfig exposing (AppConfig)
import Data.Session as Session exposing (Session)
import Html exposing (Html)
import Json.Decode as D exposing (Value)
import Layout
import Page.Home as Home
import Page.Mailbox as Mailbox
import Page.Monitor as Monitor
import Page.Status as Status
import Ports
import Route exposing (Route)
import Task
import Time
import Url exposing (Url)



-- MODEL


type alias Model =
    { layout : Layout.Model Msg
    , page : PageModel
    }


type PageModel
    = Home Home.Model
    | Mailbox Mailbox.Model
    | Monitor Monitor.Model
    | Status Status.Model


type alias InitConfig =
    { appConfig : AppConfig
    , session : Session.Persistent
    }


init : Value -> Url -> Nav.Key -> ( Model, Cmd Msg )
init configValue location key =
    let
        configDecoder =
            D.map2 InitConfig
                (D.field "app-config" AppConfig.decoder)
                (D.field "session" Session.decoder)

        session =
            case D.decodeValue configDecoder configValue of
                Ok config ->
                    Session.init key location config.appConfig config.session

                Err error ->
                    Session.initError key location (D.errorToString error)

        ( subModel, _ ) =
            Home.init session

        initModel =
            { layout = Layout.init LayoutMsg
            , page = Home subModel
            }

        route =
            session.router.fromUrl location

        ( model, cmd ) =
            changeRouteTo route initModel
    in
    ( model, Cmd.batch [ cmd, Task.perform TimeZoneLoaded Time.here ] )


type Msg
    = UrlChanged Url
    | LinkClicked UrlRequest
    | SessionUpdated (Result D.Error Session.Persistent)
    | TimeZoneLoaded Time.Zone
    | LayoutMsg Layout.Msg
    | HomeMsg Home.Msg
    | MailboxMsg Mailbox.Msg
    | MonitorMsg Monitor.Msg
    | StatusMsg Status.Msg



-- SUBSCRIPTIONS


subscriptions : Model -> Sub Msg
subscriptions model =
    Sub.batch
        [ pageSubscriptions model.page
        , Sub.map SessionUpdated sessionChange
        ]


sessionChange : Sub (Result D.Error Session.Persistent)
sessionChange =
    Ports.onSessionChange (D.decodeValue Session.decoder)


pageSubscriptions : PageModel -> Sub Msg
pageSubscriptions page =
    case page of
        Mailbox subModel ->
            Sub.map MailboxMsg (Mailbox.subscriptions subModel)

        Status subModel ->
            Sub.map StatusMsg (Status.subscriptions subModel)

        _ ->
            Sub.none



-- UPDATE


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    let
        session =
            getSession model

        ( newModel, cmd ) =
            updateMain msg model session

        newSession =
            getSession newModel
    in
    if session.persistent == newSession.persistent then
        ( newModel, cmd )

    else
        -- Store updated persistent session.
        ( newModel
        , Cmd.batch
            [ Ports.storeSession (Session.encode newSession.persistent)
            , cmd
            ]
        )


{-| Handle global/navbar related msgs.
-}
updateMain : Msg -> Model -> Session -> ( Model, Cmd Msg )
updateMain msg model session =
    case msg of
        LinkClicked req ->
            case req of
                Browser.Internal url ->
                    case url.fragment of
                        Just "" ->
                            -- Anchor tag for accessibility purposes only, already handled.
                            ( model, Cmd.none )

                        _ ->
                            ( model, Nav.pushUrl session.key (Url.toString url) )

                Browser.External url ->
                    ( model, Nav.load url )

        UrlChanged url ->
            -- Responds to new browser URL.
            if session.routing then
                changeRouteTo (session.router.fromUrl url) model

            else
                -- Skip once, but re-enable routing.
                ( applyToModelSession Session.enableRouting model
                , Cmd.none
                )

        SessionUpdated (Ok persistent) ->
            ( updateSession model { session | persistent = persistent }
            , Cmd.none
            )

        SessionUpdated (Err error) ->
            let
                flash =
                    { title = "Error decoding session"
                    , table = [ ( "Error", D.errorToString error ) ]
                    }
            in
            ( applyToModelSession (Session.showFlash flash) model
            , Cmd.none
            )

        TimeZoneLoaded zone ->
            ( updateSession model { session | zone = zone }
            , Cmd.none
            )

        LayoutMsg subMsg ->
            let
                ( layout, newSession, cmd ) =
                    Layout.update subMsg model.layout session
            in
            ( updateSession { model | layout = layout } newSession
            , cmd
            )

        _ ->
            updatePage msg model


{-| Delegate incoming messages to their respective sub-pages.
-}
updatePage : Msg -> Model -> ( Model, Cmd Msg )
updatePage msg model =
    case ( msg, model.page ) of
        ( HomeMsg subMsg, Home subModel ) ->
            Home.update subMsg subModel
                |> updateWith Home HomeMsg model

        ( MailboxMsg subMsg, Mailbox subModel ) ->
            Mailbox.update subMsg subModel
                |> updateWith Mailbox MailboxMsg model

        ( MonitorMsg subMsg, Monitor subModel ) ->
            Monitor.update subMsg subModel
                |> updateWith Monitor MonitorMsg model

        ( StatusMsg subMsg, Status subModel ) ->
            Status.update subMsg subModel
                |> updateWith Status StatusMsg model

        ( _, _ ) ->
            -- Disregard messages destined for the wrong page.
            ( model, Cmd.none )


changeRouteTo : Route -> Model -> ( Model, Cmd Msg )
changeRouteTo route model =
    let
        session =
            getSession model |> Session.clearFlash

        newModel =
            { model | layout = Layout.reset model.layout }
    in
    case route of
        Route.Home ->
            Home.init session
                |> updateWith Home HomeMsg newModel

        Route.Mailbox name ->
            Mailbox.init session name Nothing
                |> updateWith Mailbox MailboxMsg newModel

        Route.Message mailbox id ->
            Mailbox.init session mailbox (Just id)
                |> updateWith Mailbox MailboxMsg newModel

        Route.Monitor ->
            if session.config.monitorVisible then
                Monitor.init session
                    |> updateWith Monitor MonitorMsg newModel

            else
                let
                    flash =
                        { title = "Disabled route requested"
                        , table = [ ( "Error", "Monitor disabled by configuration." ) ]
                        }
                in
                ( applyToModelSession (Session.showFlash flash) newModel
                , Cmd.none
                )

        Route.Status ->
            Status.init session
                |> updateWith Status StatusMsg newModel

        Route.Unknown path ->
            -- Unknown routes display Home with an error flash.
            let
                flash =
                    { title = "Unknown route requested"
                    , table = [ ( "Path", path ) ]
                    }
            in
            Home.init (Session.showFlash flash session)
                |> updateWith Home HomeMsg newModel


getSession : Model -> Session
getSession model =
    case model.page of
        Home subModel ->
            subModel.session

        Mailbox subModel ->
            subModel.session

        Monitor subModel ->
            subModel.session

        Status subModel ->
            subModel.session


updateSession : Model -> Session -> Model
updateSession model session =
    case model.page of
        Home subModel ->
            { model | page = Home { subModel | session = session } }

        Mailbox subModel ->
            { model | page = Mailbox { subModel | session = session } }

        Monitor subModel ->
            { model | page = Monitor { subModel | session = session } }

        Status subModel ->
            { model | page = Status { subModel | session = session } }


applyToModelSession : (Session -> Session) -> Model -> Model
applyToModelSession f model =
    updateSession model (f (getSession model))


{-| Map page updates to Main Model and Msg types.
-}
updateWith :
    (subModel -> PageModel)
    -> (subMsg -> Msg)
    -> Model
    -> ( subModel, Cmd subMsg )
    -> ( Model, Cmd Msg )
updateWith toPage toMsg model ( subModel, subCmd ) =
    ( { model | page = toPage subModel }
    , Cmd.map toMsg subCmd
    )



-- VIEW


view : Model -> Document Msg
view model =
    let
        session =
            getSession model

        mailbox =
            case model.page of
                Mailbox subModel ->
                    subModel.mailboxName

                _ ->
                    ""

        framePage :
            Layout.Page
            -> (msg -> Msg)
            -> { title : String, modal : Maybe (Html msg), content : List (Html msg) }
            -> Document Msg
        framePage page toMsg { title, modal, content } =
            Document title
                [ Layout.frame
                    { model = model.layout
                    , session = session
                    , activePage = page
                    , activeMailbox = mailbox
                    , modal = Maybe.map (Html.map toMsg) modal
                    , content = List.map (Html.map toMsg) content
                    }
                ]
    in
    case model.page of
        Home subModel ->
            framePage Layout.Other HomeMsg (Home.view subModel)

        Mailbox subModel ->
            framePage Layout.Mailbox MailboxMsg (Mailbox.view subModel)

        Monitor subModel ->
            framePage Layout.Monitor MonitorMsg (Monitor.view subModel)

        Status subModel ->
            framePage Layout.Status StatusMsg (Status.view subModel)



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
