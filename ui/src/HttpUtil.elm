module HttpUtil exposing (delete, errorString, patch)

import Http


delete : String -> Http.Request ()
delete url =
    Http.request
        { method = "DELETE"
        , headers = []
        , url = url
        , body = Http.emptyBody
        , expect = Http.expectStringResponse (\_ -> Ok ())
        , timeout = Nothing
        , withCredentials = False
        }


patch : String -> Http.Body -> Http.Request ()
patch url body =
    Http.request
        { method = "PATCH"
        , headers = []
        , url = url
        , body = body
        , expect = Http.expectStringResponse (\_ -> Ok ())
        , timeout = Nothing
        , withCredentials = False
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
            "Bad HTTP status: " ++ toString res.status.code

        Http.BadPayload msg res ->
            "Bad HTTP payload: "
                ++ msg
                ++ " ("
                ++ toString res.status.code
                ++ ")"
