module Data.Date exposing (date)

import Json.Decode as Decode exposing (..)
import Time exposing (Posix)


{-| Decode a POSIX milliseconds timestamp.
-}
date : Decoder Posix
date =
    int |> andThen (Time.millisToPosix >> succeed)
