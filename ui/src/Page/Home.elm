module Page.Home exposing (Model, Msg, init, update, view)

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
    let
        cmdGreeting =
            Http.get
                { url = "/serve/greeting"
                , expect = Http.expectString GreetingLoaded
                }
    in
    ( Model "", cmdGreeting, Session.none )



-- UPDATE --


type Msg
    = GreetingLoaded (Result Http.Error String)


update : Session -> Msg -> Model -> ( Model, Cmd Msg, Session.Msg )
update session msg model =
    case msg of
        GreetingLoaded (Ok greeting) ->
            ( Model greeting, Cmd.none, Session.none )

        GreetingLoaded (Err err) ->
            ( model, Cmd.none, Session.SetFlash (HttpUtil.errorString err) )



-- VIEW --


view : Session -> Model -> { title : String, content : Html Msg }
view session model =
    { title = "Inbucket"
    , content =
        div [ id "page" ]
            [ Html.node "rendered-html"
                [ class "greeting"
                , property "content" (Encode.string model.greeting)
                ]
                []
            ]
    }
