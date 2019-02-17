module Layout exposing (Model, Msg, Page(..), frame, init, reset, update)

import Data.Session as Session exposing (Session)
import Html exposing (..)
import Html.Attributes
    exposing
        ( attribute
        , class
        , classList
        , href
        , id
        , placeholder
        , rel
        , selected
        , target
        , type_
        , value
        )
import Html.Events as Events
import Route exposing (Route)


{-| Used to highlight current page in navbar.
-}
type Page
    = Other
    | Mailbox
    | Monitor
    | Status


type alias Model msg =
    { mapMsg : Msg -> msg
    , clearFlash : msg
    , viewMailbox : String -> msg
    , menuVisible : Bool
    , recentVisible : Bool
    , mailboxName : String
    }


init : (Msg -> msg) -> msg -> (String -> msg) -> Model msg
init mapMsg clearFlash viewMailbox =
    { mapMsg = mapMsg
    , clearFlash = clearFlash
    , viewMailbox = viewMailbox
    , menuVisible = False
    , recentVisible = False
    , mailboxName = ""
    }


{-| Resets layout state, used when navigating to a new page.
-}
reset : Model msg -> Model msg
reset model =
    { model
        | menuVisible = False
        , recentVisible = False
        , mailboxName = ""
    }


type Msg
    = ToggleMenu
    | ShowRecent Bool
    | OnMailboxNameInput String


update : Msg -> Model msg -> Model msg
update msg model =
    case msg of
        ToggleMenu ->
            { model | menuVisible = not model.menuVisible }

        ShowRecent visible ->
            { model | recentVisible = visible }

        OnMailboxNameInput name ->
            { model | mailboxName = name }


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
                [ span [ class "navbar-toggle", Events.onClick (ToggleMenu |> model.mapMsg) ]
                    [ i [ class "fas fa-bars" ] [] ]
                , span [ class "navbar-brand" ]
                    [ a [ Route.href Route.Home ] [ text "@ inbucket" ] ]
                , ul [ class "main-nav", classList [ ( "active", model.menuVisible ) ] ]
                    [ if session.config.monitorVisible then
                        navbarLink Monitor Route.Monitor [ text "Monitor" ] activePage

                      else
                        text ""
                    , navbarLink Status Route.Status [ text "Status" ] activePage
                    , navbarRecent activePage activeMailbox model session
                    , li [ class "navbar-mailbox" ]
                        [ form [ Events.onSubmit (model.viewMailbox model.mailboxName) ]
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
        , frameModal modal
        , div [ class "page" ] ([ errorFlash model session.flash ] ++ content)
        , footer []
            [ div [ class "footer" ]
                [ externalLink "https://www.inbucket.org" "Inbucket"
                , text " is an open source project hosted on "
                , externalLink "https://github.com/jhillyerd/inbucket" "GitHub"
                , text "."
                ]
            ]
        ]


frameModal : Maybe (Html msg) -> Html msg
frameModal maybeModal =
    case maybeModal of
        Just modal ->
            div [ class "modal-mask" ]
                [ div [ class "modal well" ] [ modal ]
                ]

        Nothing ->
            text ""


errorFlash : Model msg -> Maybe Session.Flash -> Html msg
errorFlash controls maybeFlash =
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
                    , a [ href "#", Events.onClick controls.clearFlash ] [ text "Close" ]
                    ]
                , div [ class "flash-table" ] (List.map row flash.table)
                ]


externalLink : String -> String -> Html a
externalLink url title =
    a [ href url, target "_blank", rel "noopener" ] [ text title ]


navbarLink : Page -> Route -> List (Html a) -> Page -> Html a
navbarLink page route linkContent activePage =
    li [ classList [ ( "navbar-active", page == activePage ) ] ]
        [ a [ Route.href route ] linkContent ]


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

        dropdownExpanded =
            if model.recentVisible then
                "true"

            else
                "false"

        recentLink mailbox =
            a [ Route.href (Route.Mailbox mailbox) ] [ text mailbox ]
    in
    li
        [ class "navbar-dropdown-container"
        , classList [ ( "navbar-active", active ) ]
        , attribute "aria-haspopup" "true"
        , ariaExpanded model.recentVisible
        , Events.onMouseOver (ShowRecent True |> model.mapMsg)
        , Events.onMouseOut (ShowRecent False |> model.mapMsg)
        ]
        [ span [ class "navbar-dropdown" ]
            [ text title
            , button
                [ class "navbar-dropdown-button"
                , Events.onClick (ShowRecent (not model.recentVisible) |> model.mapMsg)
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
