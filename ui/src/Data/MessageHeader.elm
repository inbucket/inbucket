module Data.MessageHeader exposing (MessageHeader, decoder)

import Data.Date exposing (date)
import Json.Decode as Decode exposing (..)
import Json.Decode.Pipeline exposing (..)
import Time exposing (Posix)


type alias MessageHeader =
    { mailbox : String
    , id : String
    , from : String
    , to : List String
    , subject : String
    , date : Posix
    , size : Int
    , seen : Bool
    }


decoder : Decoder MessageHeader
decoder =
    succeed MessageHeader
        |> required "mailbox" string
        |> required "id" string
        |> optional "from" string ""
        |> required "to" (list string)
        |> optional "subject" string ""
        |> required "date" date
        |> required "size" int
        |> required "seen" bool
