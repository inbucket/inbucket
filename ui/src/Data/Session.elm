module Data.Session exposing
    ( Flash
    , Persistent
    , Session
    , addRecent
    , clearFlash
    , decoder
    , disableRouting
    , enableRouting
    , encode
    , init
    , initError
    , showFlash
    )

import Browser.Navigation as Nav
import Data.AppConfig as AppConfig exposing (AppConfig)
import Json.Decode as D
import Json.Decode.Pipeline exposing (optional)
import Json.Encode as E
import Route exposing (Router)
import Time
import Url exposing (Url)


type alias Session =
    { key : Nav.Key
    , host : String
    , flash : Maybe Flash
    , routing : Bool
    , router : Router
    , zone : Time.Zone
    , config : AppConfig
    , persistent : Persistent
    }


type alias Flash =
    { title : String
    , table : List ( String, String )
    }


type alias Persistent =
    { recentMailboxes : List String
    }


init : Nav.Key -> Url -> AppConfig -> Persistent -> Session
init key location config persistent =
    { key = key
    , host = location.host
    , flash = Nothing
    , routing = True
    , router = Route.newRouter config.basePath
    , zone = Time.utc
    , config = config
    , persistent = persistent
    }


initError : Nav.Key -> Url -> String -> Session
initError key location error =
    { key = key
    , host = location.host
    , flash = Just (Flash "Initialization failed" [ ( "Error", error ) ])
    , routing = True
    , router = Route.newRouter ""
    , zone = Time.utc
    , config = AppConfig.default
    , persistent = Persistent []
    }


addRecent : String -> Session -> Session
addRecent mailbox session =
    if List.head session.persistent.recentMailboxes == Just mailbox then
        session

    else
        let
            recent =
                session.persistent.recentMailboxes
                    |> List.filter ((/=) mailbox)
                    |> List.take 7
                    |> (::) mailbox

            persistent =
                session.persistent
        in
        { session | persistent = { persistent | recentMailboxes = recent } }


disableRouting : Session -> Session
disableRouting session =
    { session | routing = False }


enableRouting : Session -> Session
enableRouting session =
    { session | routing = True }


clearFlash : Session -> Session
clearFlash session =
    { session | flash = Nothing }


showFlash : Flash -> Session -> Session
showFlash flash session =
    { session | flash = Just flash }


decoder : D.Decoder Persistent
decoder =
    D.succeed Persistent
        |> optional "recentMailboxes" (D.list D.string) []


encode : Persistent -> E.Value
encode persistent =
    E.object
        [ ( "recentMailboxes", E.list E.string persistent.recentMailboxes )
        ]
