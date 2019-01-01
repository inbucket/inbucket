port module Ports exposing
    ( onSessionChange
    , storeSession
    )

import Json.Encode exposing (Value)


port onSessionChange : (Value -> msg) -> Sub msg


port storeSession : Value -> Cmd msg
