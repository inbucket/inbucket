module Effect exposing
    ( Effect
    , addRecent
    , batch
    , deleteMessage
    , disableRouting
    , enableRouting
    , getGreeting
    , getHeaderList
    , getMessage
    , getServerConfig
    , getServerMetrics
    , map
    , markMessageSeen
    , none
    , perform
    , purgeMailbox
    , showFlash
    , wrap
    )

import Api exposing (DataResult, HttpResult)
import Data.Message exposing (Message)
import Data.MessageHeader exposing (MessageHeader)
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
    = DeleteMessage (HttpResult msg) String String
    | GetGreeting (DataResult msg String)
    | GetServerConfig (DataResult msg ServerConfig)
    | GetServerMetrics (DataResult msg Metrics)
    | GetHeaderList (DataResult msg (List MessageHeader)) String
    | GetMessage (DataResult msg Message) String String
    | MarkMessageSeen (HttpResult msg) String String
    | PurgeMailbox (HttpResult msg) String


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
        DeleteMessage result mailbox id ->
            DeleteMessage (result >> f) mailbox id

        GetGreeting result ->
            GetGreeting (result >> f)

        GetServerConfig result ->
            GetServerConfig (result >> f)

        GetServerMetrics result ->
            GetServerMetrics (result >> f)

        GetHeaderList result mailbox ->
            GetHeaderList (result >> f) mailbox

        GetMessage result mailbox id ->
            GetMessage (result >> f) mailbox id

        MarkMessageSeen result mailbox id ->
            MarkMessageSeen (result >> f) mailbox id

        PurgeMailbox result mailbox ->
            PurgeMailbox (result >> f) mailbox


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
        DeleteMessage toMsg mailbox id ->
            ( session, Api.deleteMessage session toMsg mailbox id )

        GetGreeting toMsg ->
            ( session, Api.getGreeting session toMsg )

        GetServerConfig toMsg ->
            ( session, Api.getServerConfig session toMsg )

        GetServerMetrics toMsg ->
            ( session, Api.getServerMetrics session toMsg )

        GetHeaderList toMsg mailbox ->
            ( session, Api.getHeaderList session toMsg mailbox )

        GetMessage toMsg mailbox id ->
            ( session, Api.getMessage session toMsg mailbox id )

        MarkMessageSeen toMsg mailbox id ->
            ( session, Api.markMessageSeen session toMsg mailbox id )

        PurgeMailbox toMsg mailbox ->
            ( session, Api.purgeMailbox session toMsg mailbox )


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


deleteMessage : HttpResult msg -> String -> String -> Effect msg
deleteMessage toMsg mailboxName id =
    ApiEffect (DeleteMessage toMsg mailboxName id)


getGreeting : DataResult msg String -> Effect msg
getGreeting toMsg =
    ApiEffect (GetGreeting toMsg)


getHeaderList : DataResult msg (List MessageHeader) -> String -> Effect msg
getHeaderList toMsg mailboxName =
    ApiEffect (GetHeaderList toMsg mailboxName)


getServerConfig : DataResult msg ServerConfig -> Effect msg
getServerConfig toMsg =
    ApiEffect (GetServerConfig toMsg)


getServerMetrics : DataResult msg Metrics -> Effect msg
getServerMetrics toMsg =
    ApiEffect (GetServerMetrics toMsg)


getMessage : DataResult msg Message -> String -> String -> Effect msg
getMessage toMsg mailboxName id =
    ApiEffect (GetMessage toMsg mailboxName id)


markMessageSeen : HttpResult msg -> String -> String -> Effect msg
markMessageSeen toMsg mailboxName id =
    ApiEffect (MarkMessageSeen toMsg mailboxName id)


purgeMailbox : HttpResult msg -> String -> Effect msg
purgeMailbox toMsg mailboxName =
    ApiEffect (PurgeMailbox toMsg mailboxName)


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
