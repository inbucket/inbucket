module Layout exposing (Model, Msg, Page(..), frame, init, reset, update)

import Data.Session as Session exposing (Session)
import Effect exposing (Effect)
import Html
    exposing
        ( Attribute
        , Html
        , a
        , button
        , div
        , footer
        , form
        , h2
        , header
        , i
        , input
        , li
        , nav
        , pre
        , span
        , td
        , text
        , th
        , tr
        , ul
        )
import Html.Attributes
    exposing
        ( attribute
        , class
        , classList
        , href
        , placeholder
        , rel
        , target
        , type_
        , value
        )
import Html.Events as Events
import Modal
import Route
import Timer exposing (Timer)


{-| Used to highlight current page in navbar.
-}
type Page
    = Other
    | Mailbox
    | Monitor
    | Status


type alias Model msg =
    { mapMsg : Msg -> msg
    , mainMenuVisible : Bool
    , recentMenuVisible : Bool
    , recentMenuTimer : Timer
    , mailboxName : String
    }


init : (Msg -> msg) -> Model msg
init mapMsg =
    { mapMsg = mapMsg
    , mainMenuVisible = False
    , recentMenuVisible = False
    , recentMenuTimer = Timer.empty
    , mailboxName = ""
    }


{-| Resets layout state, used when navigating to a new page.
-}
reset : Model msg -> Model msg
reset model =
    { model
        | mainMenuVisible = False
        , recentMenuVisible = False
        , recentMenuTimer = Timer.cancel model.recentMenuTimer
        , mailboxName = ""
    }


type Msg
    = ClearFlash
    | MainMenuToggled
    | ModalFocused Modal.Msg
    | ModalUnfocused
    | OnMailboxNameInput String
    | OpenMailbox
    | RecentMenuMouseOver
    | RecentMenuMouseOut
    | RecentMenuTimeout Timer
    | RecentMenuToggled


update : Msg -> Model msg -> ( Model msg, Effect msg )
update msg model =
    case msg of
        ClearFlash ->
            ( model, Effect.clearFlash )

        MainMenuToggled ->
            ( { model | mainMenuVisible = not model.mainMenuVisible }, Effect.none )

        ModalFocused message ->
            ( model, Effect.focusModalResult message )

        ModalUnfocused ->
            ( model, Effect.focusModal (ModalFocused >> model.mapMsg) )

        OnMailboxNameInput name ->
            ( { model | mailboxName = name }, Effect.none )

        OpenMailbox ->
            if model.mailboxName == "" then
                ( model, Effect.none )

            else
                ( model
                , Effect.navigateRoute True (Route.Mailbox model.mailboxName)
                )

        RecentMenuMouseOver ->
            ( { model
                | recentMenuVisible = True
                , recentMenuTimer = Timer.cancel model.recentMenuTimer
              }
            , Effect.none
            )

        RecentMenuMouseOut ->
            let
                -- Keep the recent menu open for a moment even if the mouse leaves it.
                newTimer =
                    Timer.replace model.recentMenuTimer
            in
            ( { model
                | recentMenuTimer = newTimer
              }
            , Effect.schedule (RecentMenuTimeout >> model.mapMsg) newTimer 400
            )

        RecentMenuTimeout timer ->
            if timer == model.recentMenuTimer then
                ( { model
                    | recentMenuVisible = False
                    , recentMenuTimer = Timer.cancel timer
                  }
                , Effect.none
                )

            else
                -- Timer was no longer valid.
                ( model, Effect.none )

        RecentMenuToggled ->
            ( { model | recentMenuVisible = not model.recentMenuVisible }
            , Effect.none
            )


type alias State msg =
    { model : Model msg
    , session : Session
    , activePage : Page
    , activeMailbox : String
    , modal : Maybe (Html msg)
    , content : List (Html msg)
    }


frame : State msg -> Html msg
frame { model, session, activePage, activeMailbox, modal, content } =
    div [ class "app" ]
        [ header []
            [ nav [ class "navbar" ]
                [ button [ class "navbar-toggle", Events.onClick (MainMenuToggled |> model.mapMsg) ]
                    [ i [ class "fas fa-bars" ] [] ]
                , span [ class "navbar-brand" ]
                    [ a [ href <| session.router.toPath Route.Home ] [ text "@ inbucket" ] ]
                , ul [ class "main-nav", classList [ ( "active", model.mainMenuVisible ) ] ]
                    [ if session.config.monitorVisible then
                        navbarLink Monitor (session.router.toPath Route.Monitor) [ text "Monitor" ] activePage

                      else
                        text ""
                    , navbarLink Status (session.router.toPath Route.Status) [ text "Status" ] activePage
                    , navbarRecent activePage activeMailbox model session
                    , li [ class "navbar-mailbox" ]
                        [ form [ Events.onSubmit (OpenMailbox |> model.mapMsg) ]
                            [ input
                                [ type_ "text"
                                , placeholder "mailbox"
                                , value model.mailboxName
                                , Events.onInput (OnMailboxNameInput >> model.mapMsg)
                                ]
                                []
                            ]
                        ]
                    ]
                ]
            ]
        , div [ class "navbar-bg" ] [ text "" ]
        , Modal.view (ModalUnfocused |> model.mapMsg) modal
        , div [ class "page" ] (errorFlash model session.flash :: content)
        , footer []
            [ div [ class "footer" ]
                [ externalLink "https://www.inbucket.org" "Inbucket"
                , text " is an open source project hosted on "
                , externalLink "https://github.com/inbucket/inbucket" "GitHub"
                , text "."
                ]
            ]
        ]


errorFlash : Model msg -> Maybe Session.Flash -> Html msg
errorFlash model maybeFlash =
    let
        row ( heading, message ) =
            tr []
                [ th [] [ text (heading ++ ":") ]
                , td [] [ pre [] [ text message ] ]
                ]
    in
    case maybeFlash of
        Nothing ->
            text ""

        Just flash ->
            div [ class "well well-error" ]
                [ div [ class "flash-header" ]
                    [ h2 [] [ text flash.title ]
                    , a [ href "#", Events.onClick (ClearFlash |> model.mapMsg) ] [ text "Close" ]
                    ]
                , div [ class "flash-table" ] (List.map row flash.table)
                ]


externalLink : String -> String -> Html a
externalLink url title =
    a [ href url, target "_blank", rel "noopener" ] [ text title ]


navbarLink : Page -> String -> List (Html a) -> Page -> Html a
navbarLink page url linkContent activePage =
    li [ classList [ ( "navbar-active", page == activePage ) ] ]
        [ a [ href url ] linkContent ]


{-| Renders list of recent mailboxes, selecting the currently active mailbox.
-}
navbarRecent : Page -> String -> Model msg -> Session -> Html msg
navbarRecent page activeMailbox model session =
    let
        -- Active means we are viewing a specific mailbox.
        active =
            page == Mailbox

        -- Recent tab title is the name of the current mailbox when active.
        title =
            if active then
                activeMailbox

            else
                "Recent Mailboxes"

        -- Mailboxes to show in recent list, doesn't include active mailbox.
        recentMailboxes =
            if active then
                List.tail session.persistent.recentMailboxes |> Maybe.withDefault []

            else
                session.persistent.recentMailboxes

        recentLink mailbox =
            a [ href <| session.router.toPath <| Route.Mailbox mailbox ] [ text mailbox ]
    in
    li
        [ class "navbar-dropdown-container"
        , classList [ ( "navbar-active", active ) ]
        , attribute "aria-haspopup" "true"
        , ariaExpanded model.recentMenuVisible
        , Events.onMouseOver (RecentMenuMouseOver |> model.mapMsg)
        , Events.onMouseOut (RecentMenuMouseOut |> model.mapMsg)
        ]
        [ span [ class "navbar-dropdown" ]
            [ text title
            , button
                [ class "navbar-dropdown-button"
                , Events.onClick (RecentMenuToggled |> model.mapMsg)
                ]
                [ i [ class "fas fa-chevron-down" ] [] ]
            ]
        , div [ class "navbar-dropdown-content" ] (List.map recentLink recentMailboxes)
        ]


ariaExpanded : Bool -> Attribute msg
ariaExpanded value =
    attribute "aria-expanded" <|
        if value then
            "true"

        else
            "false"
