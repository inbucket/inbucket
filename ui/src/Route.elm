module Route exposing (Route(..), Router, newRouter)

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


type alias Router =
    { fromUrl : Url -> Route
    , toPath : Route -> String
    }


{-| Returns a configured Router.
-}
newRouter : String -> Router
newRouter baseUri =
    { fromUrl = fromUrl
    , toPath = toPath
    }


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


{-| Returns the Route for a given URL.
-}
fromUrl : Url -> Route
fromUrl location =
    case Parser.parse (oneOf routes) location of
        Nothing ->
            Unknown location.path

        Just route ->
            route


{-| Convert route to a URI.
-}
toPath : Route -> String
toPath page =
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
