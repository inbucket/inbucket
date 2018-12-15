module Api exposing
    ( deleteMessage
    , getGreeting
    , getHeaderList
    , getMessage
    , getServerConfig
    , getServerMetrics
    , markMessageSeen
    , purgeMailbox
    )

import Data.Message as Message exposing (Message)
import Data.MessageHeader as MessageHeader exposing (MessageHeader)
import Data.Metrics as Metrics exposing (Metrics)
import Data.ServerConfig as ServerConfig exposing (ServerConfig)
import Http
import HttpUtil
import Json.Decode as Decode
import Json.Encode as Encode
import Url.Builder


type alias DataResult msg data =
    Result Http.Error data -> msg


type alias HttpResult msg =
    Result Http.Error () -> msg


{-| Builds a public REST API URL (see wiki).
-}
apiV1Url : List String -> String
apiV1Url elements =
    Url.Builder.absolute ([ "api", "v1" ] ++ elements) []


{-| Builds an internal `serve` REST API URL; only used by this UI.
-}
serveUrl : List String -> String
serveUrl elements =
    Url.Builder.absolute ([ "serve" ] ++ elements) []


deleteMessage : HttpResult msg -> String -> String -> Cmd msg
deleteMessage msg mailboxName id =
    HttpUtil.delete msg (apiV1Url [ "mailbox", mailboxName, id ])


getHeaderList : DataResult msg (List MessageHeader) -> String -> Cmd msg
getHeaderList msg mailboxName =
    Http.get
        { url = apiV1Url [ "mailbox", mailboxName ]
        , expect = Http.expectJson msg (Decode.list MessageHeader.decoder)
        }


getGreeting : DataResult msg String -> Cmd msg
getGreeting msg =
    Http.get
        { url = serveUrl [ "greeting" ]
        , expect = Http.expectString msg
        }


getMessage : DataResult msg Message -> String -> String -> Cmd msg
getMessage msg mailboxName id =
    Http.get
        { url = serveUrl [ "m", mailboxName, id ]
        , expect = Http.expectJson msg Message.decoder
        }


getServerConfig : DataResult msg ServerConfig -> Cmd msg
getServerConfig msg =
    Http.get
        { url = serveUrl [ "status" ]
        , expect = Http.expectJson msg ServerConfig.decoder
        }


getServerMetrics : DataResult msg Metrics -> Cmd msg
getServerMetrics msg =
    Http.get
        { url = Url.Builder.absolute [ "debug", "vars" ] []
        , expect = Http.expectJson msg Metrics.decoder
        }


markMessageSeen : HttpResult msg -> String -> String -> Cmd msg
markMessageSeen msg mailboxName id =
    -- The URL tells the API which message ID to update, so we only need to indicate the
    -- desired change in the body.
    Encode.object [ ( "seen", Encode.bool True ) ]
        |> Http.jsonBody
        |> HttpUtil.patch msg (apiV1Url [ "mailbox", mailboxName, id ])


purgeMailbox : HttpResult msg -> String -> Cmd msg
purgeMailbox msg mailboxName =
    HttpUtil.delete msg (apiV1Url [ "mailbox", mailboxName ])
