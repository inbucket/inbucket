module Effect exposing (Effect, batch, map, none, perform, showFlash, wrap)

import Data.Session as Session exposing (Session, showFlash)


type Effect msg
    = None
    | Batch (List (Effect msg))
    | Command (Cmd msg)
    | SessionClearFlash
    | SessionShowFlash Session.Flash


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

        SessionClearFlash ->
            SessionClearFlash

        SessionShowFlash flash ->
            SessionShowFlash flash


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

        SessionClearFlash ->
            ( Session.clearFlash session, Cmd.none )

        SessionShowFlash flash ->
            ( Session.showFlash flash session, Cmd.none )



-- EFFECT CONSTRUCTORS


none : Effect msg
none =
    None


showFlash : Session.Flash -> Effect msg
showFlash flash =
    SessionShowFlash flash


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
