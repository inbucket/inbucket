module Page.Home exposing (Model, Msg, init, update, view)

import Html exposing (..)
import Html.Attributes exposing (..)
import Data.Session as Session exposing (Session)


-- MODEL --


type alias Model =
    {}


init : Model
init =
    {}



-- UPDATE --


type Msg
    = Msg


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    ( model, Cmd.none, Session.none )



-- VIEW --


view : Session -> Model -> Html Msg
view session model =
    div [ id "page" ]
        [ h1 [] [ text "Inbucket" ]
        , text "This is the home page"
        ]
