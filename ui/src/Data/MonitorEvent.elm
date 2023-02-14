module Data.MonitorEvent exposing (MonitorEvent(..), decoder)

import Data.MessageHeader as MessageHeader exposing (MessageHeader)
import Json.Decode exposing (Decoder, andThen, fail, field, string, succeed)
import Json.Decode.Pipeline exposing (required)


type MonitorEvent
    = MessageStored MessageHeader
    | MessageDeleted MessageHeader


decoder : Decoder MonitorEvent
decoder =
    field "variant" string
        |> andThen variantDecoder


variantDecoder : String -> Decoder MonitorEvent
variantDecoder variant =
    case variant of
        "message-deleted" ->
            succeed MessageDeleted
                |> required "header" MessageHeader.decoder

        "message-stored" ->
            succeed MessageStored
                |> required "header" MessageHeader.decoder

        unknown ->
            fail <| "Unknown variant: " ++ unknown
