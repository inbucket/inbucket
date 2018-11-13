port module Ports exposing (onSessionChange, storeSession, windowTitle)

import Data.Session exposing (Persistent)
import Json.Encode exposing (Value)


port onSessionChange : (Value -> msg) -> Sub msg


port storeSession : Persistent -> Cmd msg


port windowTitle : String -> Cmd msg
