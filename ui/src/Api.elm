module Api exposing
    ( deleteMessage
    , getGreeting
    , getHeaderList
    , getMessage
    , getServerConfig
    , getServerMetrics
    , markMessageSeen
    , purgeMailbox
    , serveUrl
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
    Result HttpUtil.Error data -> msg


type alias HttpResult msg =
    Result HttpUtil.Error () -> msg


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
    let
        context =
            { method = "GET"
            , url = apiV1Url [ "mailbox", mailboxName ]
            }
    in
    Http.get
        { url = context.url
        , expect = HttpUtil.expectJson context msg (Decode.list MessageHeader.decoder)
        }


getGreeting : DataResult msg String -> Cmd msg
getGreeting msg =
    let
        context =
            { method = "GET"
            , url = serveUrl [ "greeting" ]
            }
    in
    Http.get
        { url = context.url
        , expect = HttpUtil.expectString context msg
        }


getMessage : DataResult msg Message -> String -> String -> Cmd msg
getMessage msg mailboxName id =
    let
        context =
            { method = "GET"
            , url = serveUrl [ "mailbox", mailboxName, id ]
            }
    in
    Http.get
        { url = context.url
        , expect = HttpUtil.expectJson context msg Message.decoder
        }


getServerConfig : DataResult msg ServerConfig -> Cmd msg
getServerConfig msg =
    let
        context =
            { method = "GET"
            , url = serveUrl [ "status" ]
            }
    in
    Http.get
        { url = context.url
        , expect = HttpUtil.expectJson context msg ServerConfig.decoder
        }


getServerMetrics : DataResult msg Metrics -> Cmd msg
getServerMetrics msg =
    let
        context =
            { method = "GET"
            , url = Url.Builder.absolute [ "debug", "vars" ] []
            }
    in
    Http.get
        { url = context.url
        , expect = HttpUtil.expectJson context msg Metrics.decoder
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
