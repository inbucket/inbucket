module Data.Message exposing (..)

import Json.Decode as Decode exposing (..)
import Json.Decode.Pipeline exposing (..)


type alias Message =
    { mailbox : String
    , id : String
    , from : String
    , to : List String
    , subject : String
    , date : String
    , size : Int
    , seen : Bool
    , body : Body
    }


type alias Body =
    { text : String
    , html : String
    }


decoder : Decoder Message
decoder =
    decode Message
        |> required "mailbox" string
        |> required "id" string
        |> optional "from" string ""
        |> required "to" (list string)
        |> optional "subject" string ""
        |> required "date" string
        |> required "size" int
        |> required "seen" bool
        |> required "body" bodyDecoder


bodyDecoder : Decoder Body
bodyDecoder =
    decode Body
        |> required "text" string
        |> required "html" string
