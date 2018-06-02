module Main exposing (..)

import Data.Session as Session exposing (Session, decoder)
import Json.Decode as Decode exposing (Value)
import Html exposing (..)
import Navigation exposing (Location)
import Page.Home as Home
import Page.Mailbox as Mailbox
import Page.Monitor as Monitor
import Page.Status as Status
import Ports
import Route exposing (Route)
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


init : Value -> Location -> ( Model, Cmd Msg )
init sessionValue location =
    let
        session =
            Session.init (Session.decodeValueWithDefault sessionValue)

        model =
            { page = Home Home.init
            , session = session
            , mailboxName = ""
            }

        route =
            Route.fromLocation location
    in
        applySession (setRoute route model)


type Msg
    = SetRoute Route
    | NewRoute Route
    | UpdateSession (Result String Session.Persistent)
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


sessionChange : Sub (Result String Session.Persistent)
sessionChange =
    Ports.onSessionChange (Decode.decodeValue Session.decoder)


pageSubscriptions : Page -> Sub Msg
pageSubscriptions page =
    case page of
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
            SetRoute route ->
                -- Updates broser URL to requested route.
                ( model, Route.newUrl route, Session.none )

            NewRoute route ->
                -- Responds to new browser URL.
                if model.session.routing then
                    setRoute route model
                else
                    -- Skip once, but re-enable routing.
                    ( model, Cmd.none, Session.EnableRouting )

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
                , Route.newUrl (Route.Mailbox name)
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
    case route of
        Route.Unknown hash ->
            ( model, Cmd.none, Session.SetFlash ("Unknown route requested: " ++ hash) )

        Route.Home ->
            ( { model | page = Home Home.init }
            , Ports.windowTitle "Inbucket"
            , Session.none
            )

        Route.Mailbox name ->
            ( { model | page = Mailbox (Mailbox.init name Nothing) }
            , Cmd.map MailboxMsg (Mailbox.load name)
            , Session.none
            )

        Route.Message mailbox id ->
            ( { model | page = Mailbox (Mailbox.init mailbox (Just id)) }
            , Cmd.map MailboxMsg (Mailbox.load mailbox)
            , Session.none
            )

        Route.Monitor ->
            ( { model | page = Monitor Monitor.init }
            , Ports.windowTitle "Inbucket Monitor"
            , Session.none
            )

        Route.Status ->
            ( { model | page = Status Status.init }
            , Cmd.batch
                [ Ports.windowTitle "Inbucket Status"
                , Cmd.map StatusMsg (Status.load)
                ]
            , Session.none
            )


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


view : Model -> Html Msg
view model =
    let
        mailbox =
            case model.page of
                Mailbox subModel ->
                    subModel.name

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
        case model.page of
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



-- MAIN


main : Program Value Model Msg
main =
    Navigation.programWithFlags (Route.fromLocation >> NewRoute)
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }
