import React, { useState, useEffect } from 'react';
import {
    View,
    Text,
    StyleSheet,
    FlatList,
    TouchableOpacity,
    SafeAreaView,
    ActivityIndicator,
    Dimensions
} from 'react-native';
import { router } from 'expo-router';
import { Ionicons } from '@expo/vector-icons';
import { Colors, Spacing, BorderRadius, FontSizes } from '@/constants/theme';
import BottomNavBar from '@/components/BottomNavBar';
import { getUserReports, ReportHistory, ComplaintEvent } from '@/services/reports';

const { width } = Dimensions.get('window');

const STATUS_COLORS: Record<string, string> = {
    'submitted': '#f5a623', // Orange
    'under_review': '#e2ccf5', // Purple
    'escalated': '#ff4d4f', // Red
    'resolved': '#52c41a', // Green
};

export default function HistoryScreen() {
    const [reports, setReports] = useState<ReportHistory[]>([]);
    const [loading, setLoading] = useState(true);
    const [expandedId, setExpandedId] = useState<string | null>(null);

    useEffect(() => {
        fetchHistory();
    }, []);

    const fetchHistory = async () => {
        try {
            const data = await getUserReports();
            setReports(data);
        } catch (err) {
            console.error('Failed to fetch history:', err);
        } finally {
            setLoading(false);
        }
    };

    const onTabPress = (tab: string) => {
        if (tab === 'sos') {
            router.push('/sos' as any);
        } else if (tab === 'location') {
            router.push('/maps' as any);
        } else if (tab === 'home') {
            router.push('/home' as any);
        }
    };

    const renderTimelineItem = (event: ComplaintEvent, isLast: boolean) => (
        <View key={event.id} style={styles.timelineItem}>
            <View style={styles.timelineLeft}>
                <View style={[styles.timelineDot, { backgroundColor: STATUS_COLORS[event.status] || Colors.purple }]} />
                {!isLast && <View style={styles.timelineLine} />}
            </View>
            <View style={styles.timelineContent}>
                <Text style={styles.timelineStatus}>{event.status.replace('_', ' ').toUpperCase()}</Text>
                <Text style={styles.timelineNote}>{event.note || 'No additional notes provided.'}</Text>
                <Text style={styles.timelineTime}>
                    {new Date(event.created_at).toLocaleDateString()} at {new Date(event.created_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                </Text>
            </View>
        </View>
    );

    const renderFolder = ({ item, index }: { item: ReportHistory; index: number }) => {
        const isExpanded = expandedId === item.id;
        const folderColor = STATUS_COLORS[item.status] || Colors.purple;

        return (
            <View style={[styles.folderContainer, { marginTop: index === 0 ? 0 : -30 }]}>
                <TouchableOpacity
                    style={[styles.folderHeader, { backgroundColor: folderColor }]}
                    onPress={() => setExpandedId(isExpanded ? null : item.id)}
                    activeOpacity={0.9}
                >
                    <View style={styles.folderTab}>
                        <Text style={styles.folderTypeText}>{item.type.replace('_', ' ')}</Text>
                    </View>
                    <View style={styles.folderMain}>
                        <View style={styles.folderInfo}>
                            <Text style={styles.folderDate}>
                                {new Date(item.occurred_at).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: 'numeric' })}
                            </Text>
                            <Text style={styles.folderStatusText}>{item.status.toUpperCase()}</Text>
                        </View>
                        <Ionicons name={isExpanded ? "chevron-up" : "chevron-down"} size={24} color={Colors.bgDark} />
                    </View>
                </TouchableOpacity>

                {isExpanded && (
                    <View style={styles.folderDetails}>
                        <View style={styles.detailsGroup}>
                            <Text style={styles.detailsLabel}>Nature of event</Text>
                            <Text style={styles.detailsValue}>{item.type.replace('_', ' ')}</Text>
                        </View>

                        <View style={styles.detailsGroup}>
                            <Text style={styles.detailsLabel}>Incident Details</Text>
                            <Text style={styles.detailsValue}>{item.description || 'No details provided.'}</Text>
                        </View>

                        <View style={styles.detailsGroup}>
                            <Text style={styles.detailsLabel}>Evidence submitted</Text>
                            <Text style={styles.detailsValue}>
                                {item.evidence_ids && item.evidence_ids.length > 0
                                    ? `${item.evidence_ids.length} files attached`
                                    : 'No evidence provided'}
                            </Text>
                        </View>

                        <View style={styles.timelineSection}>
                            <Text style={styles.timelineTitle}>Update History</Text>
                            {item.events && item.events.length > 0 ? (
                                item.events.map((ev, i) => renderTimelineItem(ev, i === item.events.length - 1))
                            ) : (
                                <Text style={styles.emptyTimeline}>No updates logged yet.</Text>
                            )}
                        </View>
                    </View>
                )}
            </View>
        );
    };

    return (
        <SafeAreaView style={styles.container}>
            <View style={styles.header}>
                <Text style={styles.title}>reported incidents</Text>
            </View>

            {loading ? (
                <View style={styles.center}>
                    <ActivityIndicator size="large" color={Colors.white} />
                </View>
            ) : reports.length === 0 ? (
                <View style={styles.center}>
                    <Ionicons name="folder-open-outline" size={64} color={Colors.gray} />
                    <Text style={styles.emptyText}>No incidents reported yet</Text>
                </View>
            ) : (
                <FlatList
                    data={reports}
                    renderItem={renderFolder}
                    keyExtractor={(item) => item.id}
                    contentContainerStyle={styles.listContent}
                    showsVerticalScrollIndicator={false}
                />
            )}

            <View style={styles.bottomNav}>
                <BottomNavBar activeTab="history" onTabPress={onTabPress} />
            </View>
        </SafeAreaView>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#333333', // Dark grey matching mockup
    },
    header: {
        paddingHorizontal: Spacing.xl,
        paddingTop: Spacing.xl,
        paddingBottom: Spacing.md,
    },
    title: {
        fontSize: 28,
        fontWeight: '900',
        color: Colors.white,
        letterSpacing: -0.5,
    },
    listContent: {
        paddingHorizontal: Spacing.lg,
        paddingBottom: 160,
        paddingTop: Spacing.md,
    },
    folderContainer: {
        width: '100%',
        zIndex: 1,
    },
    folderHeader: {
        borderRadius: 24,
        borderWidth: 3,
        borderColor: Colors.bgDark,
        overflow: 'hidden',
        shadowColor: '#000',
        shadowOffset: { width: 0, height: 4 },
        shadowOpacity: 0.3,
        shadowRadius: 4,
    },
    folderTab: {
        paddingHorizontal: Spacing.lg,
        paddingVertical: 4,
        borderBottomWidth: 3,
        borderColor: Colors.bgDark,
        backgroundColor: 'rgba(0,0,0,0.1)',
        alignSelf: 'flex-start',
        borderTopLeftRadius: 20,
        borderTopRightRadius: 20,
        marginLeft: 16,
        marginTop: 8,
    },
    folderTypeText: {
        fontSize: 10,
        fontWeight: '800',
        color: Colors.bgDark,
        textTransform: 'uppercase',
    },
    folderMain: {
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'space-between',
        paddingHorizontal: Spacing.lg,
        paddingVertical: 16,
    },
    folderInfo: {
        flex: 1,
    },
    folderDate: {
        fontSize: FontSizes.md,
        fontWeight: '900',
        color: Colors.bgDark,
    },
    folderStatusText: {
        fontSize: 10,
        fontWeight: '700',
        color: 'rgba(0,0,0,0.5)',
        marginTop: 2,
    },
    folderDetails: {
        backgroundColor: Colors.white,
        borderLeftWidth: 3,
        borderRightWidth: 3,
        borderBottomWidth: 3,
        borderColor: Colors.bgDark,
        borderBottomLeftRadius: 24,
        borderBottomRightRadius: 24,
        marginTop: -12,
        paddingTop: 24,
        paddingHorizontal: Spacing.xl,
        paddingBottom: Spacing.xl,
        zIndex: -1,
    },
    detailsGroup: {
        marginBottom: Spacing.lg,
    },
    detailsLabel: {
        fontSize: 10,
        fontWeight: '800',
        color: Colors.gray,
        textTransform: 'uppercase',
        marginBottom: 4,
    },
    detailsValue: {
        fontSize: FontSizes.sm,
        fontWeight: '600',
        color: Colors.bgDark,
    },
    timelineSection: {
        marginTop: Spacing.md,
        paddingTop: Spacing.lg,
        borderTopWidth: 1,
        borderTopColor: '#EEEEEE',
    },
    timelineTitle: {
        fontSize: 12,
        fontWeight: '900',
        color: Colors.bgDark,
        textTransform: 'uppercase',
        marginBottom: Spacing.lg,
    },
    timelineItem: {
        flexDirection: 'row',
        minHeight: 60,
    },
    timelineLeft: {
        width: 20,
        alignItems: 'center',
    },
    timelineDot: {
        width: 12,
        height: 12,
        borderRadius: 6,
        borderWidth: 2,
        borderColor: Colors.bgDark,
        zIndex: 2,
    },
    timelineLine: {
        position: 'absolute',
        top: 12,
        bottom: 0,
        width: 2,
        backgroundColor: Colors.bgDark,
        zIndex: 1,
    },
    timelineContent: {
        flex: 1,
        paddingLeft: Spacing.md,
        paddingBottom: Spacing.xl,
    },
    timelineStatus: {
        fontSize: 10,
        fontWeight: '800',
        color: Colors.bgDark,
    },
    timelineNote: {
        fontSize: 12,
        color: Colors.gray,
        marginTop: 4,
    },
    timelineTime: {
        fontSize: 10,
        color: Colors.grayLight,
        marginTop: 4,
    },
    emptyTimeline: {
        fontSize: 12,
        color: Colors.grayLight,
        fontStyle: 'italic',
    },
    center: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        gap: Spacing.md,
    },
    emptyText: {
        fontSize: FontSizes.md,
        color: Colors.grayLight,
        fontWeight: '600',
    },
    bottomNav: {
        position: "absolute",
        bottom: 20,
        left: 0,
        right: 0,
        flexDirection: "row",
        justifyContent: "center",
        alignItems: "center",
        zIndex: 100,
    },
});
