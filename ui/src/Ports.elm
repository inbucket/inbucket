port module Ports exposing
    ( monitorCommand
    , monitorMessage
    , onSessionChange
    , storeSession
    )

import Json.Encode exposing (Value)


port monitorCommand : Bool -> Cmd msg


port monitorMessage : (Value -> msg) -> Sub msg


port onSessionChange : (Value -> msg) -> Sub msg


port storeSession : Value -> Cmd msg
