module Modal exposing (Msg, resetFocusCmd, updateSession, view)

import Browser.Dom as Dom
import Data.Session as Session exposing (Session)
import Html exposing (Html, div, span, text)
import Html.Attributes exposing (class, id, tabindex)
import Html.Events exposing (onFocus)
import Task


type alias Msg =
    Result Dom.Error ()


{-| Creates a command to focus the modal dialog.
-}
resetFocusCmd : (Msg -> msg) -> Cmd msg
resetFocusCmd resultMsg =
    Task.attempt resultMsg (Dom.focus domId)


{-| Updates a Session with an error Flash if the resetFocusCmd failed.
-}
updateSession : Msg -> Session -> Session
updateSession result session =
    case result of
        Ok () ->
            session

        Err (Dom.NotFound missingDomId) ->
            let
                flash =
                    { title = "DOM element not found"
                    , table = [ ( "Element ID", missingDomId ) ]
                    }
            in
            Session.showFlash flash session


view : msg -> Maybe (Html msg) -> Html msg
view unfocusedMsg maybeModal =
    case maybeModal of
        Just modal ->
            div [ class "modal-mask" ]
                [ span [ onFocus unfocusedMsg, tabindex 0 ] []
                , div [ id domId, class "modal well", tabindex -1 ] [ modal ]
                , span [ onFocus unfocusedMsg, tabindex 0 ] []
                ]

        Nothing ->
            text ""


{-| DOM ID of the modal dialog.
-}
domId : String
domId =
    "modal-dialog"
