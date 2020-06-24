module HttpUtil exposing (Error, RequestContext, delete, errorFlash, expectJson, expectString, patch)

import Data.Session as Session
import Http
import Json.Decode as Decode


type alias Error =
    { error : Http.Error
    , request : RequestContext
    }


type alias RequestContext =
    { method : String
    , url : String
    }


delete : (Result Error () -> msg) -> String -> Cmd msg
delete msg url =
    let
        context =
            { method = "DELETE"
            , url = url
            }
    in
    Http.request
        { method = context.method
        , headers = []
        , url = url
        , body = Http.emptyBody
        , expect = expectWhatever context msg
        , timeout = Nothing
        , tracker = Nothing
        }


patch : (Result Error () -> msg) -> String -> Http.Body -> Cmd msg
patch msg url body =
    let
        context =
            { method = "PATCH"
            , url = url
            }
    in
    Http.request
        { method = context.method
        , headers = []
        , url = url
        , body = body
        , expect = expectWhatever context msg
        , timeout = Nothing
        , tracker = Nothing
        }


errorFlash : Error -> Session.Flash
errorFlash error =
    let
        requestContext flash =
            { flash
                | table =
                    flash.table
                        ++ [ ( "Request URL", error.request.url )
                           , ( "HTTP Method", error.request.method )
                           ]
            }
    in
    requestContext <|
        case error.error of
            Http.BadUrl str ->
                { title = "Bad URL"
                , table = [ ( "URL", str ) ]
                }

            Http.Timeout ->
                { title = "HTTP timeout"
                , table = []
                }

            Http.NetworkError ->
                { title = "HTTP network error"
                , table = []
                }

            Http.BadStatus res ->
                { title = "HTTP response error"
                , table = [ ( "Response Code", String.fromInt res ) ]
                }

            Http.BadBody body ->
                { title = "Bad HTTP body"
                , table = [ ( "Error", body ) ]
                }


expectJson : RequestContext -> (Result Error a -> msg) -> Decode.Decoder a -> Http.Expect msg
expectJson context toMsg decoder =
    Http.expectStringResponse toMsg <|
        resolve context <|
            \string ->
                Result.mapError Decode.errorToString (Decode.decodeString decoder string)


expectString : RequestContext -> (Result Error String -> msg) -> Http.Expect msg
expectString context toMsg =
    Http.expectStringResponse toMsg (resolve context Ok)


expectWhatever : RequestContext -> (Result Error () -> msg) -> Http.Expect msg
expectWhatever context toMsg =
    Http.expectBytesResponse toMsg (resolve context (\_ -> Ok ()))


resolve : RequestContext -> (body -> Result String a) -> Http.Response body -> Result Error a
resolve context toResult response =
    case response of
        Http.BadUrl_ url ->
            Err (Error (Http.BadUrl url) context)

        Http.Timeout_ ->
            Err (Error Http.Timeout context)

        Http.NetworkError_ ->
            Err (Error Http.NetworkError context)

        Http.BadStatus_ metadata _ ->
            Err (Error (Http.BadStatus metadata.statusCode) context)

        Http.GoodStatus_ _ body ->
            Result.mapError (\x -> Error (Http.BadBody x) context) (toResult body)
