module Views.Page exposing (ActivePage(..), frame)

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


type ActivePage
    = Other
    | Mailbox
    | Monitor
    | Status


type alias FrameControls msg =
    { viewMailbox : String -> msg
    , mailboxOnInput : String -> msg
    , mailboxValue : String
    , recentOptions : List String
    , recentActive : String
    , clearFlash : msg
    }


frame : FrameControls msg -> Session -> ActivePage -> Maybe (Html msg) -> List (Html msg) -> Html msg
frame controls session page modal content =
    div [ class "app" ]
        [ header []
            [ ul [ class "navbar", attribute "role" "navigation" ]
                [ li [ class "navbar-brand" ]
                    [ a [ Route.href Route.Home ] [ text "@ inbucket" ] ]
                , navbarLink page Route.Monitor [ text "Monitor" ]
                , navbarLink page Route.Status [ text "Status" ]
                , navbarRecent page controls
                , li [ class "navbar-mailbox" ]
                    [ form [ Events.onSubmit (controls.viewMailbox controls.mailboxValue) ]
                        [ input
                            [ type_ "text"
                            , placeholder "mailbox"
                            , value controls.mailboxValue
                            , Events.onInput controls.mailboxOnInput
                            ]
                            []
                        ]
                    ]
                ]
            ]
        , div [ class "navbar-bg" ] [ text "" ]
        , frameModal modal
        , div [ class "page" ] ([ errorFlash controls session.flash ] ++ content)
        , footer []
            [ div [ class "footer" ]
                [ externalLink "https://www.inbucket.org" "Inbucket"
                , text " is an open source projected hosted at "
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


errorFlash : FrameControls msg -> Maybe Session.Flash -> Html msg
errorFlash controls maybeFlash =
    let
        row ( heading, message ) =
            pre []
                [ text heading
                , text ": "
                , text message
                ]
    in
    case maybeFlash of
        Nothing ->
            text ""

        Just flash ->
            div [ class "error" ]
                [ div [ class "flash-header" ]
                    [ h2 [] [ text flash.title ]
                    , a [ href "#", Events.onClick controls.clearFlash ] [ text "Close" ]
                    ]
                , div [ class "flash-table" ] (List.map row flash.table)
                ]


externalLink : String -> String -> Html a
externalLink url title =
    a [ href url, target "_blank", rel "noopener" ] [ text title ]


navbarLink : ActivePage -> Route -> List (Html a) -> Html a
navbarLink page route linkContent =
    li [ classList [ ( "navbar-active", isActive page route ) ] ]
        [ a [ Route.href route ] linkContent ]


{-| Renders list of recent mailboxes, selecting the currently active mailbox.
-}
navbarRecent : ActivePage -> FrameControls msg -> Html msg
navbarRecent page controls =
    let
        active =
            page == Mailbox

        -- Recent tab title is the name of the current mailbox when active.
        title =
            if active then
                controls.recentActive

            else
                "Recent Mailboxes"

        -- Mailboxes to show in recent list, doesn't include active mailbox.
        recentMailboxes =
            if active then
                List.tail controls.recentOptions |> Maybe.withDefault []

            else
                controls.recentOptions

        recentLink mailbox =
            a [ Route.href (Route.Mailbox mailbox) ] [ text mailbox ]
    in
    li
        [ class "navbar-recent"
        , classList [ ( "navbar-dropdown", True ), ( "navbar-active", active ) ]
        ]
        [ span [] [ text title ]
        , div [ class "navbar-dropdown-content" ] (List.map recentLink recentMailboxes)
        ]


isActive : ActivePage -> Route -> Bool
isActive page route =
    case ( page, route ) of
        ( Monitor, Route.Monitor ) ->
            True

        ( Status, Route.Status ) ->
            True

        _ ->
            False
