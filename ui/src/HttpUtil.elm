module HttpUtil exposing (delete, errorString, patch)

import Http


delete : (Result Http.Error () -> msg) -> String -> Cmd msg
delete msg url =
    Http.request
        { method = "DELETE"
        , headers = []
        , url = url
        , body = Http.emptyBody
        , expect = Http.expectWhatever msg
        , timeout = Nothing
        , tracker = Nothing
        }


patch : (Result Http.Error () -> msg) -> String -> Http.Body -> Cmd msg
patch msg url body =
    Http.request
        { method = "PATCH"
        , headers = []
        , url = url
        , body = body
        , expect = Http.expectWhatever msg
        , timeout = Nothing
        , tracker = Nothing
        }


errorString : Http.Error -> String
errorString error =
    case error of
        Http.BadUrl str ->
            "Bad URL: " ++ str

        Http.Timeout ->
            "HTTP timeout"

        Http.NetworkError ->
            "HTTP Network error"

        Http.BadStatus res ->
            "Bad HTTP status: " ++ String.fromInt res

        Http.BadBody msg ->
            "Bad HTTP body: " ++ msg
