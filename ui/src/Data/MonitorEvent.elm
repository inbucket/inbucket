module Data.MonitorEvent exposing (MessageID, MonitorEvent(..), decoder)

import Data.MessageHeader as MessageHeader exposing (MessageHeader)
import Json.Decode exposing (Decoder, andThen, fail, field, string, succeed)
import Json.Decode.Pipeline exposing (required)


type alias MessageID =
    { mailbox : String
    , id : String
    }


type MonitorEvent
    = MessageStored MessageHeader
    | MessageDeleted MessageID


decoder : Decoder MonitorEvent
decoder =
    field "variant" string
        |> andThen variantDecoder


variantDecoder : String -> Decoder MonitorEvent
variantDecoder variant =
    case variant of
        "message-deleted" ->
            succeed MessageDeleted
                |> required "identifier" messageIdDecoder

        "message-stored" ->
            succeed MessageStored
                |> required "header" MessageHeader.decoder

        unknown ->
            fail <| "Unknown variant: " ++ unknown


messageIdDecoder : Decoder MessageID
messageIdDecoder =
    succeed MessageID
        |> required "mailbox" string
        |> required "id" string
