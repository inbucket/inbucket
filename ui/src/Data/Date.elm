module Data.Date exposing (date)

import Json.Decode as Decode exposing (..)
import Time exposing (Posix)


{-| Decode a POSIX milliseconds timestamp. Currently faked until backend API is updated.
-}
date : Decoder Posix
date =
    succeed (Time.millisToPosix 0)
