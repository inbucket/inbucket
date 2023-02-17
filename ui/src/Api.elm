module Api exposing
    ( DataResult
    , HttpResult
    , deleteMessage
    , getGreeting
    , getHeaderList
    , getMessage
    , getServerConfig
    , getServerMetrics
    , markMessageSeen
    , monitorUri
    , purgeMailbox
    , serveUrl
    )

import Data.Message as Message exposing (Message)
import Data.MessageHeader as MessageHeader exposing (MessageHeader)
import Data.Metrics as Metrics exposing (Metrics)
import Data.ServerConfig as ServerConfig exposing (ServerConfig)
import Data.Session exposing (Session)
import Http
import HttpUtil
import Json.Decode as Decode
import Json.Encode as Encode
import String
import Url.Builder


type alias DataResult msg data =
    Result HttpUtil.Error data -> msg


type alias HttpResult msg =
    Result HttpUtil.Error () -> msg


deleteMessage : Session -> HttpResult msg -> String -> String -> Cmd msg
deleteMessage session msg mailboxName id =
    HttpUtil.delete msg (apiV1Url session [ "mailbox", mailboxName, id ])


getHeaderList : Session -> DataResult msg (List MessageHeader) -> String -> Cmd msg
getHeaderList session msg mailboxName =
    let
        context =
            { method = "GET"
            , url = apiV1Url session [ "mailbox", mailboxName ]
            }
    in
    Http.get
        { url = context.url
        , expect = HttpUtil.expectJson context msg (Decode.list MessageHeader.decoder)
        }


getGreeting : Session -> DataResult msg String -> Cmd msg
getGreeting session msg =
    let
        context =
            { method = "GET"
            , url = serveUrl session [ "greeting" ]
            }
    in
    Http.get
        { url = context.url
        , expect = HttpUtil.expectString context msg
        }


getMessage : Session -> DataResult msg Message -> String -> String -> Cmd msg
getMessage session msg mailboxName id =
    let
        context =
            { method = "GET"
            , url = serveUrl session [ "mailbox", mailboxName, id ]
            }
    in
    Http.get
        { url = context.url
        , expect = HttpUtil.expectJson context msg Message.decoder
        }


getServerConfig : Session -> DataResult msg ServerConfig -> Cmd msg
getServerConfig session msg =
    let
        context =
            { method = "GET"
            , url = serveUrl session [ "status" ]
            }
    in
    Http.get
        { url = context.url
        , expect = HttpUtil.expectJson context msg ServerConfig.decoder
        }


getServerMetrics : Session -> DataResult msg Metrics -> Cmd msg
getServerMetrics session msg =
    let
        context =
            { method = "GET"
            , url =
                Url.Builder.absolute
                    (splitBasePath session.config.basePath
                        ++ [ "debug"
                           , "vars"
                           ]
                    )
                    []
            }
    in
    Http.get
        { url = context.url
        , expect = HttpUtil.expectJson context msg Metrics.decoder
        }


markMessageSeen : Session -> HttpResult msg -> String -> String -> Cmd msg
markMessageSeen session msg mailboxName id =
    -- The URL tells the API which message ID to update, so we only need to indicate the
    -- desired change in the body.
    Encode.object [ ( "seen", Encode.bool True ) ]
        |> Http.jsonBody
        |> HttpUtil.patch msg (apiV1Url session [ "mailbox", mailboxName, id ])


monitorUri : Session -> String
monitorUri session =
    apiV2Url session [ "monitor", "messages" ]


purgeMailbox : Session -> HttpResult msg -> String -> Cmd msg
purgeMailbox session msg mailboxName =
    HttpUtil.delete msg (apiV1Url session [ "mailbox", mailboxName ])


apiV1Url : Session -> List String -> String
apiV1Url =
    apiUrl "v1"


apiV2Url : Session -> List String -> String
apiV2Url =
    apiUrl "v2"


{-| Builds a public REST API URL (see wiki).
-}
apiUrl : String -> Session -> List String -> String
apiUrl version session elements =
    Url.Builder.absolute
        (List.concat
            [ splitBasePath session.config.basePath
            , [ "api", version ]
            , elements
            ]
        )
        []


{-| Builds an internal `serve` REST API URL; only used by this UI.
-}
serveUrl : Session -> List String -> String
serveUrl session elements =
    Url.Builder.absolute
        (List.concat
            [ splitBasePath session.config.basePath
            , [ "serve" ]
            , elements
            ]
        )
        []


{-| Converts base path into a list of path elements.
-}
splitBasePath : String -> List String
splitBasePath path =
    if path == "" then
        []

    else
        let
            stripSlashes str =
                if String.startsWith "/" str then
                    stripSlashes (String.dropLeft 1 str)

                else if String.endsWith "/" str then
                    stripSlashes (String.dropRight 1 str)

                else
                    str

            newPath =
                stripSlashes path
        in
        String.split "/" newPath
