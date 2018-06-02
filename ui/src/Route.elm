module Route exposing (Route(..), fromLocation, href, modifyUrl, newUrl)

import Html exposing (Attribute)
import Html.Attributes as Attr
import Navigation exposing (Location)
import UrlParser as Url exposing ((</>), Parser, parseHash, s, string)


type Route
    = Unknown String
    | Home
    | Mailbox String
    | Message String String
    | Monitor
    | Status


matcher : Parser (Route -> a) a
matcher =
    Url.oneOf
        [ Url.map Home (s "")
        , Url.map Message (s "m" </> string </> string)
        , Url.map Mailbox (s "m" </> string)
        , Url.map Monitor (s "monitor")
        , Url.map Status (s "status")
        ]


routeToString : Route -> String
routeToString page =
    let
        pieces =
            case page of
                Unknown _ ->
                    []

                Home ->
                    []

                Mailbox name ->
                    [ "m", name ]

                Message mailbox id ->
                    [ "m", mailbox, id ]

                Monitor ->
                    [ "monitor" ]

                Status ->
                    [ "status" ]
    in
        "/#/" ++ String.join "/" pieces



-- PUBLIC HELPERS


href : Route -> Attribute msg
href route =
    Attr.href (routeToString route)


modifyUrl : Route -> Cmd msg
modifyUrl =
    routeToString >> Navigation.modifyUrl


newUrl : Route -> Cmd msg
newUrl =
    routeToString >> Navigation.newUrl


fromLocation : Location -> Route
fromLocation location =
    if String.isEmpty location.hash then
        Home
    else
        case parseHash matcher location of
            Nothing ->
                Unknown location.hash

            Just route ->
                route
