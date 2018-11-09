module Data.Date exposing (..)

import Date exposing (Date)
import Json.Decode as Decode exposing (..)


{-| Decode an ISO 8601 date
-}
date : Decoder Date
date =
    let
        convert : String -> Decoder Date
        convert raw =
            case Date.fromString raw of
                Ok date ->
                    succeed date

                Err error ->
                    fail error
    in
        string |> andThen convert
