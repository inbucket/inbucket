module Effect exposing
    ( Effect
    , addRecent
    , batch
    , disableRouting
    , enableRouting
    , getGreeting
    , getServerConfig
    , getServerMetrics
    , map
    , none
    , perform
    , showFlash
    , wrap
    )

import Api
import Data.Metrics exposing (Metrics)
import Data.ServerConfig exposing (ServerConfig)
import Data.Session as Session exposing (Session)


type Effect msg
    = None
    | Batch (List (Effect msg))
    | Command (Cmd msg)
    | ApiEffect (ApiEffect msg)
    | SessionEffect SessionEffect


type ApiEffect msg
    = GetGreeting (Api.DataResult msg String)
    | GetServerConfig (Api.DataResult msg ServerConfig)
    | GetServerMetrics (Api.DataResult msg Metrics)


type SessionEffect
    = FlashClear
    | FlashShow Session.Flash
    | RecentAdd String
    | RoutingDisable
    | RoutingEnable


{-| Packs a List of Effects into a single Effect
-}
batch : List (Effect msg) -> Effect msg
batch effects =
    Batch effects


{-| Transform message types produced by an effect.
-}
map : (a -> b) -> Effect a -> Effect b
map f effect =
    case effect of
        None ->
            None

        Batch effects ->
            Batch <| List.map (map f) effects

        Command cmd ->
            Command <| Cmd.map f cmd

        ApiEffect apiEffect ->
            ApiEffect <| mapApi f apiEffect

        SessionEffect sessionEffect ->
            SessionEffect sessionEffect


mapApi : (a -> b) -> ApiEffect a -> ApiEffect b
mapApi f effect =
    case effect of
        GetGreeting result ->
            GetGreeting <| result >> f

        GetServerConfig result ->
            GetServerConfig <| result >> f

        GetServerMetrics result ->
            GetServerMetrics <| result >> f


{-| Applies an effect by updating the session and/or producing a Cmd.
-}
perform : ( Session, Effect msg ) -> ( Session, Cmd msg )
perform ( session, effect ) =
    case Debug.log "Perform" effect of
        None ->
            ( session, Cmd.none )

        Batch effects ->
            -- TODO foldl may cause us to perform Cmds in reverse order?
            List.foldl batchPerform ( session, [] ) effects
                |> Tuple.mapSecond Cmd.batch

        Command cmd ->
            ( session, cmd )

        ApiEffect apiEffect ->
            performApi ( session, apiEffect )

        SessionEffect sessionEffect ->
            performSession ( session, sessionEffect )


performApi : ( Session, ApiEffect msg ) -> ( Session, Cmd msg )
performApi ( session, effect ) =
    case effect of
        GetGreeting toMsg ->
            ( session, Api.getGreeting session toMsg )

        GetServerConfig toMsg ->
            ( session, Api.getServerConfig session toMsg )

        GetServerMetrics toMsg ->
            ( session, Api.getServerMetrics session toMsg )


performSession : ( Session, SessionEffect ) -> ( Session, Cmd msg )
performSession ( session, effect ) =
    case effect of
        RecentAdd mailbox ->
            ( Session.addRecent mailbox session, Cmd.none )

        FlashClear ->
            ( Session.clearFlash session, Cmd.none )

        FlashShow flash ->
            ( Session.showFlash flash session, Cmd.none )

        RoutingDisable ->
            ( Session.disableRouting session, Cmd.none )

        RoutingEnable ->
            ( Session.enableRouting session, Cmd.none )



-- EFFECT CONSTRUCTORS


none : Effect msg
none =
    None


{-| Adds specified mailbox to the recently viewed list
-}
addRecent : String -> Effect msg
addRecent mailbox =
    SessionEffect (RecentAdd mailbox)


disableRouting : Effect msg
disableRouting =
    SessionEffect RoutingDisable


enableRouting : Effect msg
enableRouting =
    SessionEffect RoutingEnable


clearFlash : Effect msg
clearFlash =
    SessionEffect FlashClear


showFlash : Session.Flash -> Effect msg
showFlash flash =
    SessionEffect (FlashShow flash)


getGreeting : Api.DataResult msg String -> Effect msg
getGreeting toMsg =
    ApiEffect (GetGreeting toMsg)


getServerConfig : Api.DataResult msg ServerConfig -> Effect msg
getServerConfig toMsg =
    ApiEffect (GetServerConfig toMsg)


getServerMetrics : Api.DataResult msg Metrics -> Effect msg
getServerMetrics toMsg =
    ApiEffect (GetServerMetrics toMsg)


{-| Wrap a Cmd into an Effect. This is a temporary function to aid in the transition to the effect
pattern.
-}
wrap : Cmd msg -> Effect msg
wrap cmd =
    Command cmd



-- UTILITY


batchPerform : Effect msg -> ( Session, List (Cmd msg) ) -> ( Session, List (Cmd msg) )
batchPerform effect ( session, cmds ) =
    perform ( session, effect )
        |> Tuple.mapSecond (\cmd -> cmd :: cmds)
