module Main exposing (main)

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
import Task
import Time
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

        ( subModel, _, _ ) =
            Home.init

        initModel =
            { page = Home subModel
            , session = session
            , mailboxName = ""
            }

        route =
            Route.fromUrl location

        ( model, cmd ) =
            changeRouteTo route initModel |> updateSession
    in
    ( model, Cmd.batch [ cmd, Task.perform TimeZoneLoaded Time.here ] )


type Msg
    = UrlChanged Url
    | LinkClicked UrlRequest
    | SessionUpdated (Result D.Error Session.Persistent)
    | TimeZoneLoaded Time.Zone
    | OnMailboxNameInput String
    | ViewMailbox String
    | SessionMsg Session.Msg
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
    updateSession <|
        case msg of
            LinkClicked req ->
                case req of
                    Browser.Internal url ->
                        case url.fragment of
                            Just "" ->
                                -- Anchor tag for accessibility purposes only, already handled.
                                ( model, Cmd.none, Session.none )

                            _ ->
                                ( model
                                , Nav.pushUrl model.session.key (Url.toString url)
                                , Session.ClearFlash
                                )

                    Browser.External url ->
                        ( model, Nav.load url, Session.none )

            UrlChanged url ->
                -- Responds to new browser URL.
                if model.session.routing then
                    changeRouteTo (Route.fromUrl url) model

                else
                    -- Skip once, but re-enable routing.
                    ( model, Cmd.none, Session.EnableRouting )

            SessionMsg sessionMsg ->
                ( model, Cmd.none, sessionMsg )

            SessionUpdated (Ok persistent) ->
                let
                    session =
                        model.session
                in
                ( { model | session = { session | persistent = persistent } }
                , Cmd.none
                , Session.none
                )

            SessionUpdated (Err error) ->
                ( model
                , Cmd.none
                , Session.SetFlash ("Error decoding session:\n" ++ D.errorToString error)
                )

            TimeZoneLoaded zone ->
                let
                    session =
                        model.session
                in
                ( { model | session = { session | zone = zone } }
                , Cmd.none
                , Session.none
                )

            OnMailboxNameInput name ->
                ( { model | mailboxName = name }, Cmd.none, Session.none )

            ViewMailbox name ->
                ( { model | mailboxName = "" }
                , Route.pushUrl model.session.key (Route.Mailbox name)
                , Session.ClearFlash
                )

            _ ->
                updatePage msg model


{-| Delegates incoming messages to their respective sub-pages.
-}
updatePage : Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
updatePage msg model =
    case ( msg, model.page ) of
        ( HomeMsg subMsg, Home subModel ) ->
            Home.update model.session subMsg subModel
                |> updateWith Home HomeMsg model

        ( MailboxMsg subMsg, Mailbox subModel ) ->
            Mailbox.update model.session subMsg subModel
                |> updateWith Mailbox MailboxMsg model

        ( MonitorMsg subMsg, Monitor subModel ) ->
            Monitor.update model.session subMsg subModel
                |> updateWith Monitor MonitorMsg model

        ( StatusMsg subMsg, Status subModel ) ->
            Status.update model.session subMsg subModel
                |> updateWith Status StatusMsg model

        ( _, _ ) ->
            -- Disregard messages destined for the wrong page.
            ( model, Cmd.none, Session.none )


changeRouteTo : Route -> Model -> ( Model, Cmd Msg, Session.Msg )
changeRouteTo route model =
    let
        ( newModel, newCmd, newSession ) =
            case route of
                Route.Unknown path ->
                    ( model, Cmd.none, Session.SetFlash ("Unknown route requested: " ++ path) )

                Route.Home ->
                    Home.init
                        |> updateWith Home HomeMsg model

                Route.Mailbox name ->
                    Mailbox.init name Nothing
                        |> updateWith Mailbox MailboxMsg model

                Route.Message mailbox id ->
                    Mailbox.init mailbox (Just id)
                        |> updateWith Mailbox MailboxMsg model

                Route.Monitor ->
                    Monitor.init
                        |> updateWith Monitor MonitorMsg model

                Route.Status ->
                    Status.init
                        |> updateWith Status StatusMsg model
    in
    case model.page of
        Monitor _ ->
            -- Leaving Monitor page, shut down the web socket.
            ( newModel, Cmd.batch [ Ports.monitorCommand False, newCmd ], newSession )

        _ ->
            ( newModel, newCmd, newSession )


updateSession : ( Model, Cmd Msg, Session.Msg ) -> ( Model, Cmd Msg )
updateSession ( model, cmd, sessionMsg ) =
    let
        ( session, newCmd ) =
            Session.update sessionMsg model.session
    in
    ( { model | session = session }
    , Cmd.batch [ newCmd, cmd ]
    )


{-| Map page updates to Main Model and Msg types.
-}
updateWith :
    (subModel -> Page)
    -> (subMsg -> Msg)
    -> Model
    -> ( subModel, Cmd subMsg, Session.Msg )
    -> ( Model, Cmd Msg, Session.Msg )
updateWith toPage toMsg model ( subModel, subCmd, sessionMsg ) =
    ( { model | page = toPage subModel }
    , Cmd.map toMsg subCmd
    , sessionMsg
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
            , mailboxOnInput = OnMailboxNameInput
            , mailboxValue = model.mailboxName
            , recentOptions = model.session.persistent.recentMailboxes
            , recentActive = mailbox
            , clearFlash = SessionMsg Session.ClearFlash
            }

        framePage :
            ActivePage
            -> (msg -> Msg)
            -> { title : String, modal : Maybe (Html msg), content : Html msg }
            -> Document Msg
        framePage page toMsg { title, modal, content } =
            Document title
                [ Page.frame controls
                    model.session
                    page
                    (Maybe.map (Html.map toMsg) modal)
                    (Html.map toMsg content)
                ]
    in
    case model.page of
        Home subModel ->
            framePage Page.Other HomeMsg (Home.view model.session subModel)

        Mailbox subModel ->
            framePage Page.Mailbox MailboxMsg (Mailbox.view model.session subModel)

        Monitor subModel ->
            framePage Page.Monitor MonitorMsg (Monitor.view model.session subModel)

        Status subModel ->
            framePage Page.Status StatusMsg (Status.view model.session subModel)



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
