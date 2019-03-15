module Route exposing (Route(..), fromUrl, href, pushUrl, replaceUrl)

import Browser.Navigation as Navigation exposing (Key)
import Html exposing (Attribute)
import Html.Attributes as Attr
import Url exposing (Url)
import Url.Builder as Builder
import Url.Parser as Parser exposing ((</>), Parser, map, oneOf, s, string, top)


type Route
    = Unknown String
    | Home
    | Mailbox String
    | Message String String
    | Monitor
    | Status


{-| Routes our application handles.
-}
routes : List (Parser (Route -> a) a)
routes =
    [ map Home top
    , map Message (s "m" </> string </> string)
    , map Mailbox (s "m" </> string)
    , map Monitor (s "monitor")
    , map Status (s "status")
    ]


{-| Convert route to a URI.
-}
routeToPath : Route -> String
routeToPath page =
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
    Builder.absolute pieces []



-- PUBLIC HELPERS


href : Route -> Attribute msg
href route =
    Attr.href (routeToPath route)


replaceUrl : Key -> Route -> Cmd msg
replaceUrl key =
    routeToPath >> Navigation.replaceUrl key


pushUrl : Key -> Route -> Cmd msg
pushUrl key =
    routeToPath >> Navigation.pushUrl key


{-| Returns the Route for a given URL.
-}
fromUrl : Url -> Route
fromUrl location =
    case Parser.parse (oneOf routes) location of
        Nothing ->
            Unknown location.path

        Just route ->
            route
