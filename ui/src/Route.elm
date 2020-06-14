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
newRouter basePath =
    let
        newPath =
            prepareBasePath basePath
    in
    { fromUrl = fromUrl newPath
    , toPath = toPath newPath
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
fromUrl : String -> Url -> Route
fromUrl basePath url =
    let
        relative =
            { url | path = String.replace basePath "" url.path }
    in
    case Parser.parse (oneOf routes) relative of
        Nothing ->
            Unknown url.path

        Just route ->
            route


{-| Convert route to a URI.
-}
toPath : String -> Route -> String
toPath basePath page =
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
    basePath ++ Builder.absolute pieces []


{-| Make sure basePath starts with a slash and does not have trailing slashes.

"inbucket/" becomes "/inbucket"

-}
prepareBasePath : String -> String
prepareBasePath path =
    let
        stripSlashes str =
            if String.startsWith "/" str then
                stripSlashes (String.dropLeft 1 str)

            else if String.endsWith "/" str then
                stripSlashes (String.dropRight 1 str)

            else
                str

        newPath =
            stripSlashes path
    in
    if newPath == "" then
        ""

    else
        "/" ++ newPath
