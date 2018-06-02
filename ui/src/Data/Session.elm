module Data.Session
    exposing
        ( Session
        , Persistent
        , Msg(..)
        , decoder
        , decodeValueWithDefault
        , init
        , none
        , update
        )

import Json.Decode as Decode exposing (..)
import Json.Decode.Pipeline exposing (..)
import Navigation exposing (Location)


type alias Session =
    { host : String
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


init : Location -> Persistent -> Session
init location persistent =
    Session location.host "" True persistent


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
    decode Persistent
        |> optional "recentMailboxes" (list string) []


decodeValueWithDefault : Value -> Persistent
decodeValueWithDefault =
    Decode.decodeValue decoder >> Result.withDefault { recentMailboxes = [] }
