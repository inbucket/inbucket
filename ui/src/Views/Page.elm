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
        , type_
        , selected
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
                [ li [ id "navbar-brand" ] [ a [ Route.href Route.Home ] [ text "@ inbucket" ] ]
                , navbarLink page Route.Monitor [ text "Monitor" ]
                , navbarLink page Route.Status [ text "Status" ]
                , navbarRecent page controls
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
                [ a [ href "https://www.inbucket.org" ] [ text "Inbucket" ]
                , text " is an open source projected hosted at "
                , a [ href "https://github.com/jhillyerd/inbucket" ] [ text "GitHub" ]
                , text "."
                ]
            ]
        ]


navbarLink : ActivePage -> Route -> List (Html a) -> Html a
navbarLink page route linkContent =
    li [ classList [ ( "navbar-active", isActive page route ) ] ]
        [ a [ Route.href route ] linkContent ]


{-| Renders list of recent mailboxes, selecting the currently active mailbox.
-}
navbarRecent : ActivePage -> FrameControls msg -> Html msg
navbarRecent page controls =
    let
        recentItemLink mailbox =
            a [ Route.href (Route.Mailbox mailbox) ] [ text mailbox ]

        active =
            page == Mailbox

        -- Navbar tab title, is current mailbox when active.
        title =
            if active then
                controls.recentActive
            else
                "Recent Mailboxes"

        -- Items to show in recent list, doesn't include active mailbox.
        items =
            if active then
                List.tail controls.recentOptions |> Maybe.withDefault []
            else
                controls.recentOptions
    in
        li
            [ id "navbar-recent"
            , classList [ ( "navbar-dropdown", True ), ( "navbar-active", active ) ]
            ]
            [ span [] [ text title ]
            , div [ class "navbar-dropdown-content" ] (List.map recentItemLink items)
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
