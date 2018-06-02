module Data.MessageHeader exposing (..)

import Json.Decode as Decode exposing (..)
import Json.Decode.Pipeline exposing (..)


type alias MessageHeader =
    { mailbox : String
    , id : String
    , from : String
    , to : List String
    , subject : String
    , date : String
    , size : Int
    , seen : Bool
    }


decoder : Decoder MessageHeader
decoder =
    decode MessageHeader
        |> required "mailbox" string
        |> required "id" string
        |> optional "from" string ""
        |> required "to" (list string)
        |> optional "subject" string ""
        |> required "date" string
        |> required "size" int
        |> required "seen" bool
