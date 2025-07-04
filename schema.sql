--
-- PostgreSQL database dump
--

-- Dumped from database version 16.9
-- Dumped by pg_dump version 17.4

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: award_first_correct_submission_badge_and_update_level(); Type: FUNCTION; Schema: public; Owner: avnadmin
--

CREATE FUNCTION public.award_first_correct_submission_badge_and_update_level() RETURNS trigger
    LANGUAGE plpgsql
    AS $$

DECLARE

    unique_problems_solved INT;

    new_level INT;

BEGIN

    -- Only act if the new submission is correct

    IF NEW.correct = true THEN



        -- === First correct submission badge logic ===

        IF NOT EXISTS (

            SELECT 1

            FROM submission

            WHERE user_id = NEW.user_id

              AND correct = true

              AND submission_id <> NEW.submission_id

        ) THEN

            IF NOT EXISTS (

                SELECT 1

                FROM user_badge

                WHERE user_id = NEW.user_id

                  AND badge_id = 1

            ) THEN

                INSERT INTO user_badge(user_id, badge_id)

                VALUES (NEW.user_id, 1);

            END IF;

        END IF;



        -- === Update level based on unique problems solved ===

        SELECT COUNT(DISTINCT problem_id)

        INTO unique_problems_solved

        FROM submission

        WHERE user_id = NEW.user_id

          AND correct = true;



        new_level := FLOOR(unique_problems_solved / 2) + 1;



        -- Update user level

        UPDATE "User" 

        SET level = new_level

        WHERE user_id = NEW.user_id;

    END IF;



    RETURN NEW;

END;

$$;


ALTER FUNCTION public.award_first_correct_submission_badge_and_update_level() OWNER TO avnadmin;

--
-- Name: create_submission(text, integer, boolean, text, integer, text); Type: PROCEDURE; Schema: public; Owner: avnadmin
--

CREATE PROCEDURE public.create_submission(IN p_user_id text, IN p_problem_id integer, IN p_correct boolean, IN p_language text, IN p_time integer, IN p_submission_result text)
    LANGUAGE plpgsql
    AS $$

DECLARE

    v_points INT := 0;

    v_already_solved BOOLEAN := FALSE;

BEGIN

    -- Only check if correct

    IF p_correct THEN

        -- Check if user has already solved this problem correctly

        SELECT EXISTS (

            SELECT 1 FROM submission

            WHERE user_id = p_user_id

              AND problem_id = p_problem_id

              AND correct = true

        )

        INTO v_already_solved;



        -- If not already solved, calculate points from difficulty

        IF NOT v_already_solved THEN

            SELECT difficulty * 20

            INTO v_points

            FROM problem

            WHERE problem_id = p_problem_id;

        END IF;

    END IF;



    -- Insert new submission

    INSERT INTO submission (

        user_id,

        problem_id,

        "date",

        points,

        correct,

        language,

        "time",

        submission_result

    )

    VALUES (

        p_user_id,

        p_problem_id,

        NOW(),

        v_points,

        p_correct,

        p_language,

        p_time,

        p_submission_result

    );

    -- Add points to user only if this is the first correct

    IF v_points > 0 THEN

        UPDATE "User"

        SET points = points + v_points

        WHERE user_id = p_user_id;

    END IF;

END;

$$;


ALTER PROCEDURE public.create_submission(IN p_user_id text, IN p_problem_id integer, IN p_correct boolean, IN p_language text, IN p_time integer, IN p_submission_result text) OWNER TO avnadmin;

--
-- Name: handle_claim(); Type: FUNCTION; Schema: public; Owner: avnadmin
--

CREATE FUNCTION public.handle_claim() RETURNS trigger
    LANGUAGE plpgsql
    AS $$

DECLARE

    user_points INT;

    reward_cost INT;

    reward_inventory INT;

BEGIN

    -- Retrieve user points, reward cost, and reward inventory count

    SELECT points INTO user_points FROM "User" WHERE user_id = NEW.user_id;

    SELECT cost, inventory_count INTO reward_cost, reward_inventory FROM reward WHERE reward_id = NEW.reward_id;



    -- Ensure the user has enough points for the reward

    IF user_points < reward_cost THEN

        RAISE EXCEPTION 'User does not have enough points to claim this reward';

    END IF;



    -- Ensure the reward quantity is greater than 0

    IF reward_inventory <= 0 THEN

        RAISE EXCEPTION 'Reward quantity is not available';

    END IF;



    -- Reduce the user's points by the reward's cost

    UPDATE "User"

    SET points = points - reward_cost

    WHERE user_id = NEW.user_id;



    -- Decrease the reward's inventory count by 1

    UPDATE reward

    SET inventory_count = inventory_count - 1

    WHERE reward_id = NEW.reward_id;



    -- Return the NEW claim record

    RETURN NEW;

END;

$$;


ALTER FUNCTION public.handle_claim() OWNER TO avnadmin;

--
-- Name: update_updated_at_column(); Type: FUNCTION; Schema: public; Owner: avnadmin
--

CREATE FUNCTION public.update_updated_at_column() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$;


ALTER FUNCTION public.update_updated_at_column() OWNER TO avnadmin;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: User; Type: TABLE; Schema: public; Owner: avnadmin
--

CREATE TABLE public."User" (
    user_id character(32) NOT NULL,
    name character varying(100) NOT NULL,
    mail character varying(100) NOT NULL,
    points integer DEFAULT 0,
    level integer DEFAULT 1,
    is_admin boolean DEFAULT false
);


ALTER TABLE public."User" OWNER TO avnadmin;

--
-- Name: User_user_id_seq; Type: SEQUENCE; Schema: public; Owner: avnadmin
--

CREATE SEQUENCE public."User_user_id_seq"
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public."User_user_id_seq" OWNER TO avnadmin;

--
-- Name: User_user_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: avnadmin
--

ALTER SEQUENCE public."User_user_id_seq" OWNED BY public."User".user_id;


--
-- Name: badge; Type: TABLE; Schema: public; Owner: avnadmin
--

CREATE TABLE public.badge (
    badge_id integer NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    requirement text,
    image_url text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.badge OWNER TO avnadmin;

--
-- Name: badge_badge_id_seq; Type: SEQUENCE; Schema: public; Owner: avnadmin
--

CREATE SEQUENCE public.badge_badge_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.badge_badge_id_seq OWNER TO avnadmin;

--
-- Name: badge_badge_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: avnadmin
--

ALTER SEQUENCE public.badge_badge_id_seq OWNED BY public.badge.badge_id;


--
-- Name: claims; Type: TABLE; Schema: public; Owner: avnadmin
--

CREATE TABLE public.claims (
    claim_id integer NOT NULL,
    user_id character(32),
    reward_id integer,
    date timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.claims OWNER TO avnadmin;

--
-- Name: claims_claim_id_seq; Type: SEQUENCE; Schema: public; Owner: avnadmin
--

CREATE SEQUENCE public.claims_claim_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.claims_claim_id_seq OWNER TO avnadmin;

--
-- Name: claims_claim_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: avnadmin
--

ALTER SEQUENCE public.claims_claim_id_seq OWNED BY public.claims.claim_id;


--
-- Name: problem; Type: TABLE; Schema: public; Owner: avnadmin
--

CREATE TABLE public.problem (
    problem_id integer NOT NULL,
    question text NOT NULL,
    answer text NOT NULL,
    inputs text[],
    outputs text[],
    title character varying(255),
    difficulty integer,
    timelimit integer,
    memorylimit integer,
    tests text
);


ALTER TABLE public.problem OWNER TO avnadmin;

--
-- Name: problem_problem_id_seq; Type: SEQUENCE; Schema: public; Owner: avnadmin
--

CREATE SEQUENCE public.problem_problem_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.problem_problem_id_seq OWNER TO avnadmin;

--
-- Name: problem_problem_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: avnadmin
--

ALTER SEQUENCE public.problem_problem_id_seq OWNED BY public.problem.problem_id;


--
-- Name: reward; Type: TABLE; Schema: public; Owner: avnadmin
--

CREATE TABLE public.reward (
    reward_id integer NOT NULL,
    name character varying(100) NOT NULL,
    description text,
    inventory_count integer DEFAULT 0,
    cost integer DEFAULT 0.00 NOT NULL
);


ALTER TABLE public.reward OWNER TO avnadmin;

--
-- Name: reward_reward_id_seq; Type: SEQUENCE; Schema: public; Owner: avnadmin
--

CREATE SEQUENCE public.reward_reward_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.reward_reward_id_seq OWNER TO avnadmin;

--
-- Name: reward_reward_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: avnadmin
--

ALTER SEQUENCE public.reward_reward_id_seq OWNED BY public.reward.reward_id;


--
-- Name: submission; Type: TABLE; Schema: public; Owner: avnadmin
--

CREATE TABLE public.submission (
    submission_id integer NOT NULL,
    user_id character(32),
    problem_id integer,
    date timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    points integer DEFAULT 0,
    correct boolean DEFAULT false,
    language character varying(50),
    "time" numeric,
    submission_result text
);


ALTER TABLE public.submission OWNER TO avnadmin;

--
-- Name: submission_submission_id_seq; Type: SEQUENCE; Schema: public; Owner: avnadmin
--

CREATE SEQUENCE public.submission_submission_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.submission_submission_id_seq OWNER TO avnadmin;

--
-- Name: submission_submission_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: avnadmin
--

ALTER SEQUENCE public.submission_submission_id_seq OWNED BY public.submission.submission_id;


--
-- Name: testcases; Type: TABLE; Schema: public; Owner: avnadmin
--

CREATE TABLE public.testcases (
    testcase_id integer NOT NULL,
    problem_id integer NOT NULL,
    tin text NOT NULL,
    tout text NOT NULL
);


ALTER TABLE public.testcases OWNER TO avnadmin;

--
-- Name: testcases_testcase_id_seq; Type: SEQUENCE; Schema: public; Owner: avnadmin
--

CREATE SEQUENCE public.testcases_testcase_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER SEQUENCE public.testcases_testcase_id_seq OWNER TO avnadmin;

--
-- Name: testcases_testcase_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: avnadmin
--

ALTER SEQUENCE public.testcases_testcase_id_seq OWNED BY public.testcases.testcase_id;


--
-- Name: user_badge; Type: TABLE; Schema: public; Owner: avnadmin
--

CREATE TABLE public.user_badge (
    user_id character(32) NOT NULL,
    badge_id integer NOT NULL,
    awarded_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


ALTER TABLE public.user_badge OWNER TO avnadmin;

--
-- Name: User user_id; Type: DEFAULT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public."User" ALTER COLUMN user_id SET DEFAULT nextval('public."User_user_id_seq"'::regclass);


--
-- Name: badge badge_id; Type: DEFAULT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.badge ALTER COLUMN badge_id SET DEFAULT nextval('public.badge_badge_id_seq'::regclass);


--
-- Name: claims claim_id; Type: DEFAULT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.claims ALTER COLUMN claim_id SET DEFAULT nextval('public.claims_claim_id_seq'::regclass);


--
-- Name: problem problem_id; Type: DEFAULT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.problem ALTER COLUMN problem_id SET DEFAULT nextval('public.problem_problem_id_seq'::regclass);


--
-- Name: reward reward_id; Type: DEFAULT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.reward ALTER COLUMN reward_id SET DEFAULT nextval('public.reward_reward_id_seq'::regclass);


--
-- Name: submission submission_id; Type: DEFAULT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.submission ALTER COLUMN submission_id SET DEFAULT nextval('public.submission_submission_id_seq'::regclass);


--
-- Name: testcases testcase_id; Type: DEFAULT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.testcases ALTER COLUMN testcase_id SET DEFAULT nextval('public.testcases_testcase_id_seq'::regclass);


--
-- Name: User User_mail_key; Type: CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public."User"
    ADD CONSTRAINT "User_mail_key" UNIQUE (mail);


--
-- Name: User User_pkey; Type: CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public."User"
    ADD CONSTRAINT "User_pkey" PRIMARY KEY (user_id);


--
-- Name: badge badge_pkey; Type: CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.badge
    ADD CONSTRAINT badge_pkey PRIMARY KEY (badge_id);


--
-- Name: claims claims_pkey; Type: CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.claims
    ADD CONSTRAINT claims_pkey PRIMARY KEY (claim_id);


--
-- Name: problem problem_pkey; Type: CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.problem
    ADD CONSTRAINT problem_pkey PRIMARY KEY (problem_id);


--
-- Name: reward reward_pkey; Type: CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.reward
    ADD CONSTRAINT reward_pkey PRIMARY KEY (reward_id);


--
-- Name: submission submission_pkey; Type: CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.submission
    ADD CONSTRAINT submission_pkey PRIMARY KEY (submission_id);


--
-- Name: testcases testcases_pkey; Type: CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.testcases
    ADD CONSTRAINT testcases_pkey PRIMARY KEY (testcase_id);


--
-- Name: user_badge user_badge_pkey; Type: CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.user_badge
    ADD CONSTRAINT user_badge_pkey PRIMARY KEY (user_id, badge_id);


--
-- Name: claims handle_claim_trigger; Type: TRIGGER; Schema: public; Owner: avnadmin
--

CREATE TRIGGER handle_claim_trigger BEFORE INSERT ON public.claims FOR EACH ROW EXECUTE FUNCTION public.handle_claim();


--
-- Name: submission trigger_award_badge_and_update_level; Type: TRIGGER; Schema: public; Owner: avnadmin
--

CREATE TRIGGER trigger_award_badge_and_update_level AFTER INSERT ON public.submission FOR EACH ROW EXECUTE FUNCTION public.award_first_correct_submission_badge_and_update_level();


--
-- Name: testcases fk_problem; Type: FK CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.testcases
    ADD CONSTRAINT fk_problem FOREIGN KEY (problem_id) REFERENCES public.problem(problem_id) ON DELETE CASCADE;


--
-- Name: claims reward_id; Type: FK CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.claims
    ADD CONSTRAINT reward_id FOREIGN KEY (reward_id) REFERENCES public.reward(reward_id);


--
-- Name: submission submission_problem_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.submission
    ADD CONSTRAINT submission_problem_id_fkey FOREIGN KEY (problem_id) REFERENCES public.problem(problem_id);


--
-- Name: user_badge user_badge_badge_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.user_badge
    ADD CONSTRAINT user_badge_badge_id_fkey FOREIGN KEY (badge_id) REFERENCES public.badge(badge_id);


--
-- Name: user_badge user_badge_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.user_badge
    ADD CONSTRAINT user_badge_user_id_fkey FOREIGN KEY (user_id) REFERENCES public."User"(user_id);


--
-- Name: submission user_id; Type: FK CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.submission
    ADD CONSTRAINT user_id FOREIGN KEY (user_id) REFERENCES public."User"(user_id);


--
-- Name: claims user_id; Type: FK CONSTRAINT; Schema: public; Owner: avnadmin
--

ALTER TABLE ONLY public.claims
    ADD CONSTRAINT user_id FOREIGN KEY (user_id) REFERENCES public."User"(user_id);


--
-- PostgreSQL database dump complete
--

