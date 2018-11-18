port module Ports exposing
    ( monitorCommand
    , monitorMessage
    , onSessionChange
    , storeSession
    )

import Data.Session exposing (Persistent)
import Json.Encode exposing (Value)


port monitorCommand : Bool -> Cmd msg


port monitorMessage : (Value -> msg) -> Sub msg


port onSessionChange : (Value -> msg) -> Sub msg


port storeSession : Persistent -> Cmd msg
