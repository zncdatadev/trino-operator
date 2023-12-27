package controller

import (
	stackv1alpha1 "github.com/zncdata-labs/trino-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CoordinatorScheduler(instance *stackv1alpha1.TrinoCluster, dep *appsv1.Deployment, roleGroup *stackv1alpha1.RoleGroupCoordinatorSpec) {

	if roleGroup != nil {
		for _, roleGroup := range instance.Spec.Coordinator.RoleGroups {
			if roleGroup != nil {
				scheduler := roleGroup.Config

				if scheduler.Tolerations != nil {
					toleration := *scheduler.Tolerations

					dep.Spec.Template.Spec.Tolerations = []corev1.Toleration{
						{
							Key:               toleration.Key,
							Operator:          toleration.Operator,
							Value:             toleration.Value,
							Effect:            toleration.Effect,
							TolerationSeconds: toleration.TolerationSeconds,
						},
					}
				}

				if scheduler.Affinity != nil {
					dep.Spec.Template.Spec.Affinity = &corev1.Affinity{}
					if scheduler.Affinity.NodeAffinity != nil {
						dep.Spec.Template.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
						if scheduler.Affinity != nil && scheduler.Affinity.NodeAffinity != nil &&
							scheduler.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {

							requiredTerms := make([]corev1.NodeSelectorTerm, len(scheduler.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms))

							for i, term := range scheduler.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
								requiredTerms[i] = corev1.NodeSelectorTerm{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      term.MatchExpressions[0].Key,
											Operator: term.MatchExpressions[0].Operator,
											Values:   term.MatchExpressions[0].Values,
										},
									},
								}
							}

							if dep.Spec.Template.Spec.Affinity.NodeAffinity == nil {
								dep.Spec.Template.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
							}

							dep.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{
								NodeSelectorTerms: requiredTerms,
							}
						}

						if scheduler.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
							preferredTerms := []corev1.PreferredSchedulingTerm{}

							for _, term := range scheduler.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
								preferredTerm := corev1.PreferredSchedulingTerm{
									Weight: term.Weight,
									Preference: corev1.NodeSelectorTerm{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      term.Preference.MatchExpressions[0].Key,
												Operator: term.Preference.MatchExpressions[0].Operator,
												Values:   term.Preference.MatchExpressions[0].Values,
											},
										},
									},
								}

								preferredTerms = append(preferredTerms, preferredTerm)
							}

							dep.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferredTerms
						}
					}

					if scheduler.Affinity.PodAffinity != nil {
						dep.Spec.Template.Spec.Affinity.PodAffinity = &corev1.PodAffinity{}
						if scheduler.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
							requiredTerms := []corev1.PodAffinityTerm{}

							for _, term := range scheduler.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
								requiredTerm := corev1.PodAffinityTerm{
									Namespaces:        term.Namespaces,
									TopologyKey:       term.TopologyKey,
									NamespaceSelector: term.NamespaceSelector,
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      term.LabelSelector.MatchExpressions[0].Key,
												Operator: term.LabelSelector.MatchExpressions[0].Operator,
												Values:   term.LabelSelector.MatchExpressions[0].Values,
											},
										},
									},
								}

								requiredTerms = append(requiredTerms, requiredTerm)
							}

							dep.Spec.Template.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = requiredTerms
						}

						if scheduler.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
							preferredTerms := []corev1.WeightedPodAffinityTerm{}

							for _, term := range scheduler.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
								preferredTerm := corev1.WeightedPodAffinityTerm{
									Weight: term.Weight,
									PodAffinityTerm: corev1.PodAffinityTerm{
										Namespaces:        term.PodAffinityTerm.Namespaces,
										TopologyKey:       term.PodAffinityTerm.TopologyKey,
										NamespaceSelector: term.PodAffinityTerm.NamespaceSelector,
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      term.PodAffinityTerm.LabelSelector.MatchExpressions[0].Key,
													Operator: term.PodAffinityTerm.LabelSelector.MatchExpressions[0].Operator,
													Values:   term.PodAffinityTerm.LabelSelector.MatchExpressions[0].Values,
												},
											},
										},
									},
								}

								preferredTerms = append(preferredTerms, preferredTerm)
							}

							dep.Spec.Template.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferredTerms
						}
					}

					if scheduler.Affinity.PodAntiAffinity != nil {
						dep.Spec.Template.Spec.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
						if scheduler.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
							requiredTerms := []corev1.PodAffinityTerm{}

							for _, term := range scheduler.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
								requiredTerm := corev1.PodAffinityTerm{
									Namespaces:        term.Namespaces,
									TopologyKey:       term.TopologyKey,
									NamespaceSelector: term.NamespaceSelector,
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      term.LabelSelector.MatchExpressions[0].Key,
												Operator: term.LabelSelector.MatchExpressions[0].Operator,
												Values:   term.LabelSelector.MatchExpressions[0].Values,
											},
										},
									},
								}

								requiredTerms = append(requiredTerms, requiredTerm)
							}

							dep.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = requiredTerms
						}

						if scheduler.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
							preferredTerms := []corev1.WeightedPodAffinityTerm{}

							for _, term := range scheduler.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
								preferredTerm := corev1.WeightedPodAffinityTerm{
									Weight: term.Weight,
									PodAffinityTerm: corev1.PodAffinityTerm{
										Namespaces:        term.PodAffinityTerm.Namespaces,
										TopologyKey:       term.PodAffinityTerm.TopologyKey,
										NamespaceSelector: term.PodAffinityTerm.NamespaceSelector,
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      term.PodAffinityTerm.LabelSelector.MatchExpressions[0].Key,
													Operator: term.PodAffinityTerm.LabelSelector.MatchExpressions[0].Operator,
													Values:   term.PodAffinityTerm.LabelSelector.MatchExpressions[0].Values,
												},
											},
										},
									},
								}

								preferredTerms = append(preferredTerms, preferredTerm)
							}

							dep.Spec.Template.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferredTerms
						}
					}
				}
			}
		}
	}
}

func WorkerScheduler(instance *stackv1alpha1.TrinoCluster, dep *appsv1.Deployment, roleGroup *stackv1alpha1.RoleGroupsWorkerSpec) {

	if roleGroup != nil {
		for _, roleGroup := range instance.Spec.Worker.RoleGroups {
			if roleGroup != nil {
				scheduler := roleGroup.Config
				// 使用 scheduler

				if scheduler.Tolerations != nil {
					toleration := *scheduler.Tolerations

					dep.Spec.Template.Spec.Tolerations = []corev1.Toleration{
						{
							Key:               toleration.Key,
							Operator:          toleration.Operator,
							Value:             toleration.Value,
							Effect:            toleration.Effect,
							TolerationSeconds: toleration.TolerationSeconds,
						},
					}
				}

				if scheduler.Affinity != nil {
					dep.Spec.Template.Spec.Affinity = &corev1.Affinity{}
					if scheduler.Affinity.NodeAffinity != nil {
						dep.Spec.Template.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
						if scheduler.Affinity != nil && scheduler.Affinity.NodeAffinity != nil &&
							scheduler.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {

							requiredTerms := make([]corev1.NodeSelectorTerm, len(scheduler.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms))

							for i, term := range scheduler.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
								requiredTerms[i] = corev1.NodeSelectorTerm{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{
											Key:      term.MatchExpressions[0].Key,
											Operator: term.MatchExpressions[0].Operator,
											Values:   term.MatchExpressions[0].Values,
										},
									},
								}
							}

							if dep.Spec.Template.Spec.Affinity.NodeAffinity == nil {
								dep.Spec.Template.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
							}

							dep.Spec.Template.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{
								NodeSelectorTerms: requiredTerms,
							}
						}

						if scheduler.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
							preferredTerms := []corev1.PreferredSchedulingTerm{}

							for _, term := range scheduler.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
								preferredTerm := corev1.PreferredSchedulingTerm{
									Weight: term.Weight,
									Preference: corev1.NodeSelectorTerm{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      term.Preference.MatchExpressions[0].Key,
												Operator: term.Preference.MatchExpressions[0].Operator,
												Values:   term.Preference.MatchExpressions[0].Values,
											},
										},
									},
								}

								preferredTerms = append(preferredTerms, preferredTerm)
							}

							dep.Spec.Template.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferredTerms
						}
					}

					if scheduler.Affinity.PodAffinity != nil {
						dep.Spec.Template.Spec.Affinity.PodAffinity = &corev1.PodAffinity{}
						if scheduler.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
							requiredTerms := []corev1.PodAffinityTerm{}

							for _, term := range scheduler.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
								requiredTerm := corev1.PodAffinityTerm{
									Namespaces:        term.Namespaces,
									TopologyKey:       term.TopologyKey,
									NamespaceSelector: term.NamespaceSelector,
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      term.LabelSelector.MatchExpressions[0].Key,
												Operator: term.LabelSelector.MatchExpressions[0].Operator,
												Values:   term.LabelSelector.MatchExpressions[0].Values,
											},
										},
									},
								}

								requiredTerms = append(requiredTerms, requiredTerm)
							}

							dep.Spec.Template.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = requiredTerms
						}

						if scheduler.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
							preferredTerms := []corev1.WeightedPodAffinityTerm{}

							for _, term := range scheduler.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
								preferredTerm := corev1.WeightedPodAffinityTerm{
									Weight: term.Weight,
									PodAffinityTerm: corev1.PodAffinityTerm{
										Namespaces:        term.PodAffinityTerm.Namespaces,
										TopologyKey:       term.PodAffinityTerm.TopologyKey,
										NamespaceSelector: term.PodAffinityTerm.NamespaceSelector,
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      term.PodAffinityTerm.LabelSelector.MatchExpressions[0].Key,
													Operator: term.PodAffinityTerm.LabelSelector.MatchExpressions[0].Operator,
													Values:   term.PodAffinityTerm.LabelSelector.MatchExpressions[0].Values,
												},
											},
										},
									},
								}

								preferredTerms = append(preferredTerms, preferredTerm)
							}

							dep.Spec.Template.Spec.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferredTerms
						}
					}

					if scheduler.Affinity.PodAntiAffinity != nil {
						dep.Spec.Template.Spec.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
						if scheduler.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
							requiredTerms := []corev1.PodAffinityTerm{}

							for _, term := range scheduler.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
								requiredTerm := corev1.PodAffinityTerm{
									Namespaces:        term.Namespaces,
									TopologyKey:       term.TopologyKey,
									NamespaceSelector: term.NamespaceSelector,
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      term.LabelSelector.MatchExpressions[0].Key,
												Operator: term.LabelSelector.MatchExpressions[0].Operator,
												Values:   term.LabelSelector.MatchExpressions[0].Values,
											},
										},
									},
								}

								requiredTerms = append(requiredTerms, requiredTerm)
							}

							dep.Spec.Template.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = requiredTerms
						}

						if scheduler.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution != nil {
							preferredTerms := []corev1.WeightedPodAffinityTerm{}

							for _, term := range scheduler.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
								preferredTerm := corev1.WeightedPodAffinityTerm{
									Weight: term.Weight,
									PodAffinityTerm: corev1.PodAffinityTerm{
										Namespaces:        term.PodAffinityTerm.Namespaces,
										TopologyKey:       term.PodAffinityTerm.TopologyKey,
										NamespaceSelector: term.PodAffinityTerm.NamespaceSelector,
										LabelSelector: &metav1.LabelSelector{
											MatchExpressions: []metav1.LabelSelectorRequirement{
												{
													Key:      term.PodAffinityTerm.LabelSelector.MatchExpressions[0].Key,
													Operator: term.PodAffinityTerm.LabelSelector.MatchExpressions[0].Operator,
													Values:   term.PodAffinityTerm.LabelSelector.MatchExpressions[0].Values,
												},
											},
										},
									},
								}

								preferredTerms = append(preferredTerms, preferredTerm)
							}

							dep.Spec.Template.Spec.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferredTerms
						}
					}
				}
			}

		}
	}
}
