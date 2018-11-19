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
    }


frame : FrameControls msg -> Session -> ActivePage -> Html msg -> Html msg
frame controls session page content =
    div [ id "app" ]
        [ header []
            [ ul [ class "navbar", attribute "role" "navigation" ]
                [ li [ id "navbar-brand" ]
                    [ a [ Route.href session.key Route.Home ] [ text "@ inbucket" ] ]
                , navbarLink session page Route.Monitor [ text "Monitor" ]
                , navbarLink session page Route.Status [ text "Status" ]
                , navbarRecent session page controls
                , li [ id "navbar-mailbox" ]
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
            , div [] [ text ("Status: " ++ session.flash) ]
            ]
        , div [ id "navbg" ] [ text "" ]
        , content
        , footer []
            [ div [ id "footer" ]
                [ externalLink "https://www.inbucket.org" "Inbucket"
                , text " is an open source projected hosted at "
                , externalLink "https://github.com/jhillyerd/inbucket" "GitHub"
                , text "."
                ]
            ]
        ]


externalLink : String -> String -> Html a
externalLink url title =
    a [ href url, target "_blank", rel "noopener" ] [ text title ]


navbarLink : Session -> ActivePage -> Route -> List (Html a) -> Html a
navbarLink session page route linkContent =
    li [ classList [ ( "navbar-active", isActive page route ) ] ]
        [ a [ Route.href session.key route ] linkContent ]


{-| Renders list of recent mailboxes, selecting the currently active mailbox.
-}
navbarRecent : Session -> ActivePage -> FrameControls msg -> Html msg
navbarRecent session page controls =
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
            a [ Route.href session.key (Route.Mailbox mailbox) ] [ text mailbox ]
    in
    li
        [ id "navbar-recent"
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
