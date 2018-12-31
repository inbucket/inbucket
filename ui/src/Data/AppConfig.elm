module Data.AppConfig exposing (AppConfig, decoder, default)

import Json.Decode as D
import Json.Decode.Pipeline as P


type alias AppConfig =
    { monitorVisible : Bool
    }


decoder : D.Decoder AppConfig
decoder =
    D.succeed AppConfig
        |> P.required "monitor-visible" D.bool


default : AppConfig
default =
    AppConfig True
