module Page.Home exposing (Model, Msg, init, update, view)

import Api
import Data.Session as Session exposing (Session)
import Html exposing (..)
import Html.Attributes exposing (..)
import Http
import HttpUtil
import Json.Encode as Encode
import Ports



-- MODEL --


type alias Model =
    { greeting : String }


init : ( Model, Cmd Msg, Session.Msg )
init =
    ( Model "", Api.getGreeting GreetingLoaded, Session.none )



-- UPDATE --


type Msg
    = GreetingLoaded (Result HttpUtil.Error String)


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    case msg of
        GreetingLoaded (Ok greeting) ->
            ( Model greeting, Cmd.none, Session.none )

        GreetingLoaded (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorFlash err) )



-- VIEW --


view : Session -> Model -> { title : String, modal : Maybe (Html msg), content : List (Html Msg) }
view session model =
    { title = "Inbucket"
    , modal = Nothing
    , content =
        [ Html.node "rendered-html"
            [ class "greeting"
            , property "content" (Encode.string model.greeting)
            ]
            []
        ]
    }
