import unittest
from pathlib import Path
from tempfile import TemporaryDirectory

from decision_engine import CaseSnapshot, DecisionEngine, DecisionRequest
from decision_engine.agent_team import LocalCodeAgent, SupervisorAgentTeam, Verifier
from decision_engine.local_code import LocalCodeInspector, LocalRepoConfig
from decision_engine.models import AgentReport, ContextLedgerItem, DecisionResponse, KnowledgeCandidate, ToolPlan, ToolSpec


class DecisionEngineTest(unittest.TestCase):
    def test_kline_missing_fields_asks_user(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="kline"),
                entities={"symbol": "BTCUSDT"},
            )
        )

        self.assertEqual(response.action, "ask_user")
        self.assertIn("interval", response.missing_fields)
        self.assertIn("abnormal_time", response.missing_fields)
        self.assertIsNotNone(response.verification)
        self.assertEqual(response.verification.tool_count, 0)

    def test_kline_complete_fields_plans_tools(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="kline"),
                entities={
                    "symbol": "BTCUSDT",
                    "interval": "1m",
                    "abnormal_time": "2026-05-21T20:00:00+08:00",
                    "issue_type": "price_mismatch",
                },
                max_tool_calls=2,
            )
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertEqual([item.tool_name for item in response.tool_plan], ["get_internal_kline", "get_external_kline_compare"])
        self.assertEqual([item.agent_name for item in response.agent_reports], ["supervisor", "knowledge_agent", "kline_agent"])
        self.assertTrue(response.verification.accepted)
        self.assertEqual(response.verification.tool_budget, 2)

    def test_available_tools_filter_plan(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="asset"),
                entities={
                    "user_id": "u_1",
                    "asset_symbol": "USDT",
                    "abnormal_time": "2026-05-21T20:00:00+08:00",
                    "issue_type": "balance_mismatch",
                },
                available_tools=[ToolSpec(name="get_asset_events")],
            )
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertEqual([item.tool_name for item in response.tool_plan], ["get_asset_events"])
        self.assertEqual(response.agent_reports[-1].agent_name, "asset_agent")

    def test_health_food_plans_readonly_tools_after_uid(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_hf", issue_domain="health_food", issue_type="每日推荐缺失"),
                entities={"uid": "hf-user-001", "issue_type": "每日推荐缺失", "recommendation_date": "2026-05-24"},
                available_tools=[
                    ToolSpec(name="get_health_food_user_profile"),
                    ToolSpec(name="get_health_food_recommendation_status"),
                    ToolSpec(name="search_logs_by_service"),
                ],
            )
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertEqual(
            [item.tool_name for item in response.tool_plan],
            ["get_health_food_user_profile", "get_health_food_recommendation_status", "search_logs_by_service"],
        )
        self.assertEqual(response.agent_reports[-1].agent_name, "health_food_agent")

    def test_llm_decision_advisor_is_verified_before_tool_invocation(self) -> None:
        class FakeAdvisor:
            name = "llm_decision_agent"

            def evaluate(self, request, agent_reports, default_proposal):
                return AgentReport(
                    agent_name=self.name,
                    action="invoke_tools",
                    reason="advisor prefers quota first but also hallucinated one unsafe tool",
                    tool_plan=[
                        ToolPlan(tool_name="delete_user", reason="unsafe hallucination", arguments={"uid": "hf-user-001"}),
                        ToolPlan(tool_name="get_health_food_ai_quota", reason="readonly quota check", arguments={"uid": "hf-user-001"}),
                    ],
                    confidence=0.8,
                )

        engine = DecisionEngine(SupervisorAgentTeam(decision_advisor=FakeAdvisor()))
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_hf", issue_domain="health_food", issue_type="token 消耗异常"),
                entities={"uid": "hf-user-001", "issue_type": "token 消耗异常"},
                available_tools=[
                    ToolSpec(name="get_health_food_ai_quota"),
                    ToolSpec(name="get_health_food_recommendation_status"),
                ],
            )
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertEqual([item.tool_name for item in response.tool_plan], ["get_health_food_ai_quota"])
        self.assertIn("llm_decision_agent", [item.agent_name for item in response.agent_reports])
        self.assertIn("unavailable_tool=delete_user", response.verification.violations)

    def test_supervisor_tracks_context_ledger_without_expanding_context(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_hf", issue_domain="health_food", issue_type="每日推荐缺失"),
                entities={"uid": "hf-user-001", "issue_type": "每日推荐缺失"},
                available_tools=[ToolSpec(name="get_health_food_recommendation_status")],
                context_ledger=[
                    ContextLedgerItem(
                        ledger_type="tool_evidence",
                        ledger_key="get_health_food_recommendation_status",
                        summary="上一次查询未发现推荐记录",
                        evidence_refs=[{"ref_type": "gateway_tool_call", "ref_id": "tc_1"}],
                        source_agent="tool:get_health_food_recommendation_status",
                    )
                ],
            )
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertIn("context_ledger_items=1", response.agent_reports[0].observations)
        self.assertIn("context_ledger_snapshot_loaded", response.verification.checks)

    def test_health_food_requires_uid_before_tools(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_hf", issue_domain="health_food", issue_type="每日推荐缺失"),
                entities={"issue_type": "每日推荐缺失"},
                available_tools=[ToolSpec(name="get_health_food_recommendation_status")],
            )
        )

        self.assertEqual(response.action, "ask_user")
        self.assertEqual(response.missing_fields[0], "user_id_or_uid")
        self.assertEqual(response.agent_reports[-1].agent_name, "health_food_agent")

    def test_asset_requires_user_or_account(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="asset"),
                entities={
                    "asset_symbol": "USDT",
                    "abnormal_time": "2026-05-21T20:00:00+08:00",
                    "issue_type": "balance_mismatch",
                },
            )
        )

        self.assertEqual(response.action, "ask_user")
        self.assertEqual(response.missing_fields[0], "user_id_or_account_id")
        self.assertEqual(response.agent_reports[-1].agent_name, "asset_agent")

    def test_high_confidence_knowledge_can_answer_directly(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="kline"),
                entities={
                    "symbol": "BTCUSDT",
                    "interval": "1m",
                    "abnormal_time": "2026-05-21T20:00:00+08:00",
                    "issue_type": "known_sop",
                },
                knowledge_candidates=[
                    KnowledgeCandidate(
                        title="历史 SOP",
                        confidence=0.93,
                        observed_case_count=3,
                        requires_realtime_check=False,
                        source="knowledge:1",
                    )
                ],
            )
        )

        self.assertEqual(response.action, "answer_from_knowledge")
        self.assertEqual(response.knowledge_source, "knowledge:1")
        self.assertEqual([item.agent_name for item in response.agent_reports], ["supervisor", "knowledge_agent"])
        self.assertTrue(response.verification.accepted)
        self.assertEqual(response.verification.tool_count, 0)

    def test_low_signal_greeting_does_not_hit_knowledge_or_gateway(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_greeting", original_text="你好"),
                knowledge_candidates=[
                    KnowledgeCandidate(
                        title="不应命中的历史经验",
                        confidence=0.99,
                        observed_case_count=9,
                        requires_realtime_check=False,
                        source="knowledge:999",
                    )
                ],
                available_tools=[ToolSpec(name="search_logs_by_service")],
            )
        )

        self.assertEqual(response.action, "ask_user")
        self.assertEqual(response.missing_fields, ["problem_description"])
        self.assertEqual([item.agent_name for item in response.agent_reports], ["supervisor", "intake_agent"])
        self.assertTrue(response.verification.accepted)
        self.assertEqual(response.verification.tool_count, 0)

    def test_realtime_knowledge_still_invokes_tools(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="kline"),
                entities={
                    "symbol": "BTCUSDT",
                    "interval": "1m",
                    "abnormal_time": "2026-05-21T20:00:00+08:00",
                    "issue_type": "price_mismatch",
                },
                knowledge_candidates=[
                    KnowledgeCandidate(
                        title="历史 SOP",
                        confidence=0.94,
                        observed_case_count=4,
                        requires_realtime_check=True,
                    )
                ],
            )
        )

        self.assertEqual(response.action, "invoke_tools")

    def test_decision_engine_realtime_gate_uses_problem_text(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(
                    case_no="case_1",
                    issue_domain="health_food",
                    issue_type="餐食数据异常",
                    original_text="health-food uid hf-1 2026-05-23 请查真实数据，确认推荐来源餐食是否错配",
                ),
                entities={"uid": "hf-1", "issue_type": "餐食数据异常", "recommendation_date": "2026-05-23"},
                knowledge_candidates=[
                    KnowledgeCandidate(
                        title="历史 SOP",
                        confidence=0.96,
                        observed_case_count=5,
                        requires_realtime_check=False,
                        source="knowledge:1",
                    )
                ],
                available_tools=[ToolSpec(name="get_health_food_recommendation_status")],
            )
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertIn("realtime_check_required=True", response.agent_reports[1].observations)

    def test_unknown_domain_uses_fallback_agent(self) -> None:
        engine = DecisionEngine()
        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_1", issue_domain="ops"),
                entities={"service_name": "market-api", "trace_id": "trace_1"},
                max_tool_calls=2,
            )
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertEqual([item.tool_name for item in response.tool_plan], ["search_logs_by_service", "get_recent_deployments"])
        self.assertEqual(response.agent_reports[-1].agent_name, "fallback_agent")

    def test_verifier_filters_unavailable_and_truncates_tools(self) -> None:
        verifier = Verifier()
        request = DecisionRequest(
            case=CaseSnapshot(case_no="case_1", issue_domain="asset"),
            available_tools=[ToolSpec(name="get_asset_events")],
            max_tool_calls=1,
        )
        proposal = DecisionResponse(
            action="invoke_tools",
            reason="specialist proposal",
            tool_plan=[
                ToolPlan(tool_name="get_asset_snapshot", reason="unavailable"),
                ToolPlan(tool_name="get_asset_events", reason="allowed"),
                ToolPlan(tool_name="get_similar_cases", reason="over budget"),
            ],
        )

        response = verifier.verify(
            request,
            proposal,
            [AgentReport(agent_name="asset_agent", action="invoke_tools", reason="test")],
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertEqual([item.tool_name for item in response.tool_plan], ["get_asset_events"])
        self.assertIn("unavailable_tool=get_asset_snapshot", response.verification.violations)
        self.assertIn("tool_plan_truncated_by_budget", response.verification.violations)

    def test_local_code_debug_inspects_allowlisted_repo(self) -> None:
        with TemporaryDirectory() as tmpdir:
            repo = Path(tmpdir)
            source_file = repo / "src/main/java/com/example/RecommendationJob.java"
            source_file.parent.mkdir(parents=True)
            source_file.write_text(
                "class RecommendationJob {\n"
                "  private IFoodService foodService;\n"
                "  void run() {\n"
                "    foodService.generateDailyFoodRecommendWithFingerprint(uid, meals);\n"
                "    String mealDataFingerprint = \"stale\";\n"
                "  }\n"
                "}\n",
                encoding="utf-8",
            )
            interface_file = repo / "src/main/java/com/example/IFoodService.java"
            interface_file.write_text(
                "interface IFoodService {\n"
                "  boolean generateDailyFoodRecommendWithFingerprint(Long uid, List meals);\n"
                "}\n",
                encoding="utf-8",
            )
            impl_file = repo / "src/main/java/com/example/FoodServiceImpl.java"
            impl_file.write_text(
                "class FoodServiceImpl implements IFoodService {\n"
                "  boolean generateDailyFoodRecommendWithFingerprint(Long uid, List meals) { return true; }\n"
                "}\n",
                encoding="utf-8",
            )
            secret_file = repo / "src/main/resources/application-prod.yml"
            secret_file.parent.mkdir(parents=True, exist_ok=True)
            secret_file.write_text("recommendation:\n  token: should_not_be_returned\n", encoding="utf-8")
            inspector = LocalCodeInspector(
                repos={
                    "health-food": LocalRepoConfig(
                        service_name="health-food",
                        repo_path=repo,
                        allowed_globs=("src/main/java/**", "src/main/resources/**"),
                    )
                }
            )
            engine = DecisionEngine(SupervisorAgentTeam(local_code_agent=LocalCodeAgent(inspector)))

            response = engine.plan(
                DecisionRequest(
                    case=CaseSnapshot(
                        case_no="case_local_code",
                        issue_domain="health_food",
                        issue_type="recommendation_missing",
                        original_text="health-food 今日推荐没生成，怀疑 mealDataFingerprint 没刷新",
                    ),
                    entities={
                        "debug_local_code": "true",
                        "gateway_evidence_status": "insufficient",
                        "service_name": "health-food",
                        "suspect_area": "recommendation mealDataFingerprint",
                    },
                )
            )

            self.assertEqual(response.action, "local_code_inspection")
            self.assertTrue(response.verification.accepted)
            local_report = response.agent_reports[-1]
            self.assertEqual(local_report.agent_name, "local_code_agent")
            self.assertEqual(local_report.evidence[0]["file_path"], "src/main/java/com/example/RecommendationJob.java")
            evidence_text = str(local_report.evidence)
            self.assertIn("symbols", evidence_text)
            self.assertIn("call_edges", evidence_text)
            self.assertIn("implement_relations", evidence_text)
            self.assertIn("generateDailyFoodRecommendWithFingerprint", evidence_text)
            self.assertIn("'receiver_type': 'IFoodService'", evidence_text)
            self.assertIn("resolved_symbols", evidence_text)
            self.assertIn("FoodServiceImpl.generateDailyFoodRecommendWithFingerprint", evidence_text)
            self.assertIn("IFoodService.generateDailyFoodRecommendWithFingerprint", evidence_text)
            self.assertIn("resolved_call_edge_count=1", local_report.observations)
            self.assertIn("implement_relation_count=1", local_report.observations)
            self.assertIn(
                "analysis_modes=keyword,language_structure_tree,symbol_index,call_graph,cross_module_call_resolution,interface_implementation",
                local_report.observations,
            )
            self.assertIn("analysis_backends=lightweight,cross_module_resolver", local_report.observations)
            self.assertNotIn("application-prod.yml", str(local_report.evidence))
            self.assertNotIn("should_not_be_returned", str(local_report.evidence))
            self.assertIn("bounded_code_excerpt", response.verification.checks)
            self.assertIn("secret_line_masking", response.verification.checks)
            first_hit = local_report.evidence[0]
            self.assertEqual(first_hit["primary_symbol"]["name"], "RecommendationJob.run")
            self.assertEqual(first_hit["primary_symbol"]["kind"], "method")
            self.assertIn("line_range", first_hit)
            self.assertIn("code_excerpt", first_hit)
            self.assertIn("suspect_reasons", first_hit)
            self.assertIn("follow_up_checks", first_hit)
            self.assertIn("foodService.generateDailyFoodRecommendWithFingerprint(uid, meals);", str(first_hit["code_excerpt"]))

    def test_local_code_config_accepts_semantic_backend_slots(self) -> None:
        config = LocalRepoConfig.from_dict(
            "health-food",
            {
                "repo_path": "/tmp/health-food",
                "analysis_backend": "lsp",
                "lsif_path": "/tmp/health-food/index.lsif",
                "lsp_command": ["jdtls", "--stdio"],
            },
        )

        self.assertEqual(config.analysis_backend, "lsp")
        self.assertEqual(str(config.lsif_path), "/tmp/health-food/index.lsif")
        self.assertEqual(config.lsp_command, ("jdtls", "--stdio"))

    def test_local_code_debug_uses_python_ast_calls(self) -> None:
        with TemporaryDirectory() as tmpdir:
            repo = Path(tmpdir)
            source_file = repo / "service.py"
            source_file.write_text(
                "class RecommendationService:\n"
                "    def run(self):\n"
                "        return generate_daily_food_recommend()\n"
                "\n"
                "def generate_daily_food_recommend():\n"
                "    return True\n",
                encoding="utf-8",
            )
            inspector = LocalCodeInspector(
                repos={
                    "health-food": LocalRepoConfig(
                        service_name="health-food",
                        repo_path=repo,
                        allowed_globs=("**/*.py",),
                    )
                }
            )
            engine = DecisionEngine(SupervisorAgentTeam(local_code_agent=LocalCodeAgent(inspector)))

            response = engine.plan(
                DecisionRequest(
                    case=CaseSnapshot(case_no="case_python_code", issue_domain="health_food"),
                    entities={
                        "debug_local_code": "true",
                        "gateway_evidence_status": "insufficient",
                        "service_name": "health-food",
                        "suspect_area": "generate_daily_food_recommend",
                    },
                )
            )

            self.assertEqual(response.action, "local_code_inspection")
            evidence_text = str(response.agent_reports[-1].evidence)
            self.assertIn("RecommendationService.run", evidence_text)
            self.assertIn("generate_daily_food_recommend", evidence_text)
            self.assertIn("call_edges", evidence_text)

    def test_local_code_debug_requires_gateway_insufficient_status(self) -> None:
        inspector = LocalCodeInspector(repos={})
        engine = DecisionEngine(SupervisorAgentTeam(local_code_agent=LocalCodeAgent(inspector)))

        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_no_code", issue_domain="ops"),
                entities={
                    "debug_local_code": "true",
                    "gateway_evidence_status": "sufficient",
                    "service_name": "health-food",
                },
                max_tool_calls=1,
            )
        )

        self.assertEqual(response.action, "invoke_tools")
        self.assertEqual(response.agent_reports[-1].agent_name, "fallback_agent")

    def test_local_code_debug_without_mapping_needs_human(self) -> None:
        inspector = LocalCodeInspector(repos={})
        engine = DecisionEngine(SupervisorAgentTeam(local_code_agent=LocalCodeAgent(inspector)))

        response = engine.plan(
            DecisionRequest(
                case=CaseSnapshot(case_no="case_no_mapping", issue_domain="health_food"),
                entities={
                    "debug_local_code": "true",
                    "gateway_evidence_status": "insufficient",
                    "service_name": "health-food",
                    "suspect_area": "recommendation",
                },
            )
        )

        self.assertEqual(response.action, "need_human")
        self.assertIn("local_repo_mapping_missing", response.agent_reports[-1].risks)

    def test_local_code_debug_skips_symlink_outside_repo(self) -> None:
        with TemporaryDirectory() as tmpdir:
            base = Path(tmpdir)
            repo = base / "repo"
            outside = base / "outside"
            repo_source = repo / "src/main/java/com/example"
            outside.mkdir()
            repo_source.mkdir(parents=True)
            outside_file = outside / "SecretRecommendation.java"
            outside_file.write_text("class SecretRecommendation { String token = \"hidden\"; }\n", encoding="utf-8")
            link = repo_source / "SecretRecommendation.java"
            try:
                link.symlink_to(outside_file)
            except OSError:
                self.skipTest("symlink is not available on this filesystem")

            inspector = LocalCodeInspector(
                repos={
                    "health-food": LocalRepoConfig(
                        service_name="health-food",
                        repo_path=repo,
                        allowed_globs=("src/main/java/**",),
                    )
                }
            )
            engine = DecisionEngine(SupervisorAgentTeam(local_code_agent=LocalCodeAgent(inspector)))

            response = engine.plan(
                DecisionRequest(
                    case=CaseSnapshot(case_no="case_symlink", issue_domain="health_food"),
                    entities={
                        "debug_local_code": "true",
                        "gateway_evidence_status": "insufficient",
                        "service_name": "health-food",
                        "suspect_area": "SecretRecommendation token",
                    },
                )
            )

            self.assertEqual(response.action, "need_human")
            self.assertEqual(response.agent_reports[-1].evidence, [])
            self.assertIn("skipped_denied_files=1", response.agent_reports[-1].observations)


if __name__ == "__main__":
    unittest.main()
