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
    { session : Session
    , greeting : String
    }


init : Session -> ( Model, Cmd Msg, Session.Msg )
init session =
    ( Model session "", Api.getGreeting GreetingLoaded, Session.none )



-- UPDATE --


type Msg
    = GreetingLoaded (Result HttpUtil.Error String)


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    case msg of
        GreetingLoaded (Ok greeting) ->
            ( { model | greeting = greeting }, Cmd.none, Session.none )

        GreetingLoaded (Err err) ->
            ( { model | session = Session.showFlash (HttpUtil.errorFlash err) model.session }
            , Cmd.none
            , Session.none
            )



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
