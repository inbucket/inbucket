module Data.MessageHeader exposing (Label, MessageHeader, labelDecoder, decoder)

import Data.Date exposing (date)
import Json.Decode exposing (Decoder, bool, int, list, string, succeed)
import Json.Decode.Pipeline exposing (optional, required)
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
    , labels : List Label
    }


type alias Label =
    { name : String
    , description : String
    , color : String
    }


decoder : Decoder MessageHeader
decoder =
    succeed MessageHeader
        |> required "mailbox" string
        |> required "id" string
        |> optional "from" string ""
        |> required "to" (list string)
        |> optional "subject" string ""
        |> required "posix-millis" date
        |> required "size" int
        |> required "seen" bool
        |> optional "labels" (list labelDecoder) []


labelDecoder : Decoder Label
labelDecoder =
    succeed Label
        |> required "name" string
        |> optional "description" string ""
        |> optional "color" string ""
