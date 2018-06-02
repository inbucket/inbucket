module Data.Metrics exposing (..)

import Json.Decode as Decode exposing (..)
import Json.Decode.Pipeline exposing (..)


type alias Metrics =
    { sysMem : Int
    , heapSize : Int
    , heapUsed : Int
    , heapObjects : Int
    , goRoutines : Int
    , webSockets : Int
    , smtpConnOpen : Int
    , smtpConnTotal : Int
    , smtpConnHist : List Int
    , smtpReceivedTotal : Int
    , smtpReceivedHist : List Int
    , smtpErrorsTotal : Int
    , smtpErrorsHist : List Int
    , smtpWarnsTotal : Int
    , smtpWarnsHist : List Int
    , retentionDeletesTotal : Int
    , retentionDeletesHist : List Int
    , retainedCount : Int
    , retainedCountHist : List Int
    , retainedSize : Int
    , retainedSizeHist : List Int
    }


decoder : Decoder Metrics
decoder =
    decode Metrics
        |> requiredAt [ "memstats", "Sys" ] int
        |> requiredAt [ "memstats", "HeapSys" ] int
        |> requiredAt [ "memstats", "HeapAlloc" ] int
        |> requiredAt [ "memstats", "HeapObjects" ] int
        |> requiredAt [ "goroutines" ] int
        |> requiredAt [ "http", "WebSocketConnectsCurrent" ] int
        |> requiredAt [ "smtp", "ConnectsCurrent" ] int
        |> requiredAt [ "smtp", "ConnectsTotal" ] int
        |> requiredAt [ "smtp", "ConnectsHist" ] decodeIntList
        |> requiredAt [ "smtp", "ReceivedTotal" ] int
        |> requiredAt [ "smtp", "ReceivedHist" ] decodeIntList
        |> requiredAt [ "smtp", "ErrorsTotal" ] int
        |> requiredAt [ "smtp", "ErrorsHist" ] decodeIntList
        |> requiredAt [ "smtp", "WarnsTotal" ] int
        |> requiredAt [ "smtp", "WarnsHist" ] decodeIntList
        |> requiredAt [ "retention", "DeletesTotal" ] int
        |> requiredAt [ "retention", "DeletesHist" ] decodeIntList
        |> requiredAt [ "retention", "RetainedCurrent" ] int
        |> requiredAt [ "retention", "RetainedHist" ] decodeIntList
        |> requiredAt [ "retention", "RetainedSize" ] int
        |> requiredAt [ "retention", "SizeHist" ] decodeIntList


{-| Decodes Inbuckets hacky comma-separated-int JSON strings.
-}
decodeIntList : Decoder (List Int)
decodeIntList =
    map (String.split "," >> List.map (String.toInt >> Result.withDefault 0)) string
