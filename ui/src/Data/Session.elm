module Data.Session exposing
    ( Flash
    , Msg(..)
    , Persistent
    , Session
    , addRecent
    , clearFlash
    , decodeValueWithDefault
    , decoder
    , disableRouting
    , enableRouting
    , init
    , none
    , showFlash
    , update
    )

import Browser.Navigation as Nav
import Html exposing (Html)
import Json.Decode as D
import Json.Decode.Pipeline exposing (..)
import Json.Encode as E
import Ports
import Time
import Url exposing (Url)


type alias Session =
    { key : Nav.Key
    , host : String
    , flash : Maybe Flash
    , routing : Bool
    , zone : Time.Zone
    , persistent : Persistent
    }


type alias Flash =
    { title : String
    , table : List ( String, String )
    }


type alias Persistent =
    { recentMailboxes : List String
    }


type Msg
    = None


init : Nav.Key -> Url -> Persistent -> Session
init key location persistent =
    { key = key
    , host = location.host
    , flash = Nothing
    , routing = True
    , zone = Time.utc
    , persistent = persistent
    }


update : Msg -> Session -> ( Session, Cmd a )
update msg session =
    let
        newSession =
            case msg of
                None ->
                    session
    in
    if session.persistent == newSession.persistent then
        -- No change
        ( newSession, Cmd.none )

    else
        ( newSession
        , Ports.storeSession (encode newSession.persistent)
        )


none : Msg
none =
    None


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


decodeValueWithDefault : D.Value -> Persistent
decodeValueWithDefault =
    D.decodeValue decoder >> Result.withDefault { recentMailboxes = [] }


encode : Persistent -> E.Value
encode persistent =
    E.object
        [ ( "recentMailboxes", E.list E.string persistent.recentMailboxes )
        ]
