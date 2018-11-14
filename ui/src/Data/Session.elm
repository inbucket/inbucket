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
import Json.Decode exposing (..)
import Json.Decode.Pipeline exposing (..)
import Url exposing (Url)


type alias Session =
    { key : Nav.Key
    , host : String
    , flash : String
    , routing : Bool
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
    Session key location.host "" True persistent


update : Msg -> Session -> Session
update msg session =
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


none : Msg
none =
    None


decoder : Decoder Persistent
decoder =
    succeed Persistent
        |> optional "recentMailboxes" (list string) []


decodeValueWithDefault : Value -> Persistent
decodeValueWithDefault =
    decodeValue decoder >> Result.withDefault { recentMailboxes = [] }
