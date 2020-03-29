module Data.Date exposing (date)

import Json.Decode exposing (Decoder, int, map)
import Time exposing (Posix)


{-| Decode a POSIX milliseconds timestamp.
-}
date : Decoder Posix
date =
    int |> map Time.millisToPosix
