module Effect exposing (Effect, batch, map, none, perform, wrap)

import Data.Session exposing (Session)


type Effect msg
    = None
    | Batch (List (Effect msg))
    | Command (Cmd msg)


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


none : Effect msg
none =
    None


{-| Applies an effect by updating the session and/or producing a Cmd.
-}
perform : ( Session, Effect msg ) -> ( Session, Cmd msg )
perform ( session, effect ) =
    case effect of
        None ->
            ( session, Cmd.none )

        Batch effects ->
            -- TODO foldl may cause us to perform Cmds in reverse order?
            List.foldl batchPerform ( session, [] ) effects
                |> Tuple.mapSecond Cmd.batch

        Command cmd ->
            ( session, cmd )


batchPerform : Effect msg -> ( Session, List (Cmd msg) ) -> ( Session, List (Cmd msg) )
batchPerform effect ( session, cmds ) =
    perform ( session, effect )
        |> Tuple.mapSecond (\cmd -> cmd :: cmds)


{-| Wrap a Cmd into an Effect. This is a temporary function to aid in the transition to the effect
pattern.
-}
wrap : Cmd msg -> Effect msg
wrap cmd =
    Command cmd
