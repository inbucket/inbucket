module Data.Metrics exposing (Metrics, decodeIntList, decoder)

import Data.Date exposing (date)
import Json.Decode exposing (Decoder, int, map, string, succeed)
import Json.Decode.Pipeline exposing (requiredAt)
import Time exposing (Posix)


type alias Metrics =
    { startTime : Posix
    , sysMem : Int
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
    , scanCompleted : Posix
    }


decoder : Decoder Metrics
decoder =
    succeed Metrics
        |> requiredAt [ "startMillis" ] date
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
        |> requiredAt [ "retention", "ScanCompletedMillis" ] date


{-| Decodes Inbuckets hacky comma-separated-int JSON strings.
-}
decodeIntList : Decoder (List Int)
decodeIntList =
    string
        |> map (String.split ",")
        |> map (List.map (String.toInt >> Maybe.withDefault 0))
