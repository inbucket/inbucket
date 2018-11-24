module Data.Session exposing
    ( Msg(..)
    , Persistent
    , Session
    , decodeValueWithDefault
    , decoder
    , init
    , none
    , update
    )

import Browser.Navigation as Nav
import Json.Decode as D
import Json.Decode.Pipeline exposing (..)
import Json.Encode as E
import Ports
import Time
import Url exposing (Url)


type alias Session =
    { key : Nav.Key
    , host : String
    , flash : String
    , routing : Bool
    , zone : Time.Zone
    , persistent : Persistent
    }


type alias Persistent =
    { recentMailboxes : List String
    }


type Msg
    = None
    | SetFlash String
    | ClearFlash
    | DisableRouting
    | EnableRouting
    | AddRecent String


init : Nav.Key -> Url -> Persistent -> Session
init key location persistent =
    { key = key
    , host = location.host
    , flash = ""
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

                SetFlash flash ->
                    { session | flash = flash }

                ClearFlash ->
                    { session | flash = "" }

                DisableRouting ->
                    { session | routing = False }

                EnableRouting ->
                    { session | routing = True }

                AddRecent mailbox ->
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
